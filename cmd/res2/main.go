package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	redisclient "github.com/redis/go-redis/v9"

	emailadapter "github.com/wecratfs/commerce/internal/adapters/email"
	"github.com/wecratfs/commerce/internal/adapters/meilisearch"
	metricsadapter "github.com/wecratfs/commerce/internal/adapters/metrics"
	"github.com/wecratfs/commerce/internal/adapters/postgres"
	razorpayadapter "github.com/wecratfs/commerce/internal/adapters/razorpay"
	redisadapter "github.com/wecratfs/commerce/internal/adapters/redis"
	storageadapter "github.com/wecratfs/commerce/internal/adapters/storage"
	applicationauth "github.com/wecratfs/commerce/internal/application/auth"
	applicationcatalog "github.com/wecratfs/commerce/internal/application/catalog"
	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	applicationpayment "github.com/wecratfs/commerce/internal/application/payment"
	applicationprivacy "github.com/wecratfs/commerce/internal/application/privacy"
	"github.com/wecratfs/commerce/internal/config"
	"github.com/wecratfs/commerce/internal/ports"
	apiTransport "github.com/wecratfs/commerce/internal/transport/api"
	healthTransport "github.com/wecratfs/commerce/internal/transport/health"
	webTransport "github.com/wecratfs/commerce/internal/transport/web"
	webhookTransport "github.com/wecratfs/commerce/internal/transport/webhook"
	"github.com/wecratfs/commerce/internal/worker"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	cfg := config.FromEnv()

	startupContext, startupCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer startupCancel()
	pool, err := pgxpool.New(startupContext, cfg.DatabaseURL)
	if err != nil {
		logger.Error("create PostgreSQL pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(startupContext); err != nil {
		logger.Error("connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	if err := postgres.ApplyMigrations(startupContext, pool); err != nil {
		logger.Error("apply PostgreSQL migrations", "error", err)
		os.Exit(1)
	}
	redisClient := redisclient.NewClient(&redisclient.Options{Addr: cfg.RedisAddress})
	defer redisClient.Close()
	rateLimiter := redisadapter.NewRateLimiter(redisClient)

	catalogService := applicationcatalog.NewService(postgres.NewProductRepository(pool))
	searchRepository := postgres.NewProductSearchRepository(pool)
	searchEngine := meilisearch.NewSearchEngineWithKey(cfg.MeiliAddress, cfg.MeiliAPIKey)
	searchCache := redisadapter.NewCache(redisClient)
	searchMetrics := metricsadapter.NewSearchMetrics()
	searchService := applicationcatalog.NewSearchServiceWithEngineAndObserver(searchRepository, searchCache, searchEngine, searchMetrics)
	passwordHasher := applicationauth.NewArgon2idHasher()
	authService := applicationauth.NewService(postgres.NewUserRepository(pool), passwordHasher)
	sessionService := applicationauth.NewSessionService(redisadapter.NewSessionStore(redisClient))
	var emailSender ports.EmailSender
	if cfg.SMTPAddress != "" && cfg.SMTPFrom != "" {
		emailSender = emailadapter.NewSMTP(cfg.SMTPAddress, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
	} else {
		logger.Info("account recovery email delivery is disabled; configure WECRATFS_SMTP_ADDR and WECRATFS_SMTP_FROM to enable it")
	}
	recoveryService := applicationauth.NewRecoveryService(postgres.NewAuthActionRepository(pool), passwordHasher, emailSender, cfg.PublicURL)
	commerceRepository := postgres.NewCommerceRepository(pool)
	roleAuthorizer := postgres.NewRoleAuthorizer(pool)
	commerceCatalog := applicationcommerce.NewCatalogServiceWithRoles(commerceRepository, roleAuthorizer)
	sellerStaffService := applicationcommerce.NewSellerStaffService(postgres.NewSellerStaffRepository(pool), roleAuthorizer)
	inventoryService := applicationcommerce.NewInventoryService(commerceRepository, roleAuthorizer)
	sellerOrderService := applicationcommerce.NewSellerOrderService(commerceRepository, roleAuthorizer)
	sellerReportService := applicationcommerce.NewSellerReportService(commerceRepository, roleAuthorizer)
	sellerAdminService := applicationcommerce.NewSellerAdminService(commerceRepository, roleAuthorizer)
	auditService := applicationcommerce.NewAuditService(commerceRepository, roleAuthorizer)
	roleAdminService := applicationauth.NewRoleAdminService(postgres.NewRoleAdministrationRepository(pool), roleAuthorizer)
	adminOperationsService := applicationcommerce.NewAdminOperationsService(commerceRepository, roleAuthorizer)
	brandDirectoryService := applicationcommerce.NewBrandDirectoryService(commerceRepository, roleAuthorizer)
	customerDirectoryService := applicationauth.NewCustomerDirectoryService(postgres.NewUserRepository(pool), roleAuthorizer)
	financeService := applicationcommerce.NewFinanceService(commerceRepository, roleAuthorizer)
	supportService := applicationcommerce.NewSupportService(commerceRepository, roleAuthorizer)
	returnService := applicationcommerce.NewReturnService(commerceRepository, roleAuthorizer)
	cartService := applicationcommerce.NewCartService(commerceRepository)
	orderService := applicationcommerce.NewOrderService(commerceRepository)
	privacyService := applicationprivacy.NewService(postgres.NewPrivacyRepository(pool))
	paymentGateway := razorpayadapter.NewGateway(cfg.RazorpayKeyID, cfg.RazorpayKeySecret, cfg.RazorpayWebhookSecret, cfg.RazorpayBaseURL)
	paymentRepository := postgres.NewPaymentRepository(pool)
	paymentService := applicationpayment.NewService(paymentGateway, paymentRepository)
	refundService := applicationpayment.NewRefundService(paymentGateway, paymentRepository)
	mediaStorage, storageErr := storageadapter.NewS3(cfg.S3Address, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3Secure)
	if storageErr != nil {
		logger.Warn("S3 media adapter unavailable", "error", storageErr)
	} else if err := mediaStorage.EnsureBucket(startupContext); err != nil {
		logger.Warn("S3 media bucket unavailable", "error", err)
	}
	mediaService := applicationcommerce.NewMediaService(mediaStorage, postgres.NewMediaRepository(pool))
	paymentOrderService := applicationpayment.NewOrderService(paymentGateway, paymentRepository)
	webHandler, err := webTransport.NewFullHandlerWithPaymentConfigMediaRefundAndRecovery(catalogService, authService, sessionService, commerceCatalog, cartService, orderService, paymentOrderService, searchService, privacyService, mediaService, refundService, recoveryService, cfg.SecureCookies, cfg.RazorpayKeyID)
	if err != nil {
		logger.Error("load web templates", "error", err)
		os.Exit(1)
	}
	webHandler.SetSellerStaffService(sellerStaffService)
	webHandler.SetInventoryService(inventoryService)
	webHandler.SetSellerOrderService(sellerOrderService)
	webHandler.SetSellerReportService(sellerReportService)
	webHandler.SetSellerAdminService(sellerAdminService)
	webHandler.SetAuditService(auditService)
	webHandler.SetRoleAdminService(roleAdminService)
	webHandler.SetAdminOperationsService(adminOperationsService)
	webHandler.SetBrandDirectoryService(brandDirectoryService)
	webHandler.SetCustomerDirectoryService(customerDirectoryService)
	webHandler.SetFinanceService(financeService)
	webHandler.SetSupportService(supportService)
	webHandler.SetReturnService(returnService)
	apiHandler := apiTransport.NewCompleteHandlerWithMediaPaymentRefundAndRecovery(catalogService, searchService, authService, sessionService, cartService, orderService, privacyService, mediaService, paymentOrderService, refundService, recoveryService, cfg.SecureCookies)
	backgroundWorker, err := worker.New(1, 16)
	if err != nil {
		logger.Error("configure background worker", "error", err)
		os.Exit(1)
	}
	workerContext, stopWorker := context.WithCancel(context.Background())
	backgroundWorker.Start(workerContext)
	productIndexProcessor := applicationcatalog.NewProductIndexProcessorWithCache(postgres.NewOutboxRepository(pool), searchRepository, searchEngine, searchCache)
	go worker.RunPeriodic(workerContext, time.Minute, func(ctx context.Context) error {
		cleanupContext, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		_, indexErr := productIndexProcessor.Process(cleanupContext, 10)
		if indexErr != nil {
			logger.Warn("product index processing failed", "error", indexErr)
		}
		return indexErr
	})
	go worker.RunPeriodic(workerContext, time.Minute, func(ctx context.Context) error {
		cleanupContext, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_, cleanupErr := commerceRepository.ReleaseExpiredReservations(cleanupContext, 100)
		if cleanupErr != nil {
			logger.Warn("release expired reservations failed", "error", cleanupErr)
		}
		return cleanupErr
	})
	go worker.RunPeriodic(workerContext, time.Minute, func(ctx context.Context) error {
		cleanupContext, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		_, cleanupErr := mediaService.CleanupExpired(cleanupContext, 100)
		if cleanupErr != nil && !errors.Is(cleanupErr, applicationcommerce.ErrInvalidMedia) {
			logger.Warn("expired media cleanup failed", "error", cleanupErr)
		}
		return cleanupErr
	})
	defer stopWorker()
	healthHandler := healthTransport.NewHandlerWithMetrics(map[string]healthTransport.Probe{
		"postgresql":  func(ctx context.Context) error { return postgres.Ping(ctx, pool) },
		"redis":       tcpProbe(cfg.RedisAddress),
		"meilisearch": tcpProbe(cfg.MeiliAddress),
		"s3":          tcpProbe(cfg.S3Address),
	}, func() any { return searchMetrics.Snapshot() })

	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", healthHandler.Live)
	mux.HandleFunc("/health/ready", healthHandler.Ready)
	mux.HandleFunc("/health/metrics", healthHandler.Metrics)
	mux.HandleFunc("/api/v1/products", apiHandler.Products)
	mux.HandleFunc("/api/v1/search", apiHandler.Search)
	mux.HandleFunc("/api/v1/auth/register", apiHandler.Register)
	mux.HandleFunc("/api/v1/auth/login", apiHandler.Login)
	mux.HandleFunc("/api/v1/auth/logout", apiHandler.Logout)
	mux.HandleFunc("/api/v1/auth/password-reset/request", apiHandler.PasswordResetRequest)
	mux.HandleFunc("/api/v1/auth/password-reset/confirm", apiHandler.PasswordResetConfirm)
	mux.HandleFunc("/api/v1/auth/email-verification/request", apiHandler.EmailVerificationRequest)
	mux.HandleFunc("/api/v1/auth/email-verification/confirm", apiHandler.EmailVerificationConfirm)
	mux.HandleFunc("/api/v1/cart", apiHandler.Cart)
	mux.HandleFunc("/api/v1/cart/add", apiHandler.CartAdd)
	mux.HandleFunc("/api/v1/wishlist", apiHandler.Wishlist)
	mux.HandleFunc("/api/v1/wishlist/add", apiHandler.WishlistAdd)
	mux.HandleFunc("/api/v1/wishlist/", apiHandler.WishlistRemove)
	mux.HandleFunc("/api/v1/checkout", apiHandler.Checkout)
	mux.HandleFunc("/api/v1/orders", apiHandler.Orders)
	mux.HandleFunc("/api/v1/orders/", apiHandler.Order)
	mux.HandleFunc("/api/v1/privacy", apiHandler.PrivacyCenter)
	mux.HandleFunc("/api/v1/privacy/consent", apiHandler.PrivacyConsent)
	mux.HandleFunc("/api/v1/privacy/requests", apiHandler.PrivacyRequest)
	mux.HandleFunc("/api/v1/privacy/delete", apiHandler.PrivacyDelete)
	mux.HandleFunc("/api/v1/seller/products/media-url", apiHandler.MediaUploadURL)
	mux.HandleFunc("/api/v1/seller/products/media-finalize", apiHandler.MediaFinalize)
	mux.HandleFunc("/api/v1/seller/products/media-delete", apiHandler.MediaDelete)
	mux.Handle("/webhooks/razorpay", webhookTransport.NewRazorpayHandler(paymentService))
	mux.Handle("/", webHandler)

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           requestID(authRateLimit(mux, rateLimiter)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverError := make(chan error, 1)
	go func() {
		logger.Info("WeCratfs server listening", "address", cfg.Address)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverError <- err
		}
	}()

	shutdownSignal, stopSignal := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignal()
	select {
	case <-shutdownSignal.Done():
		stopWorker()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := backgroundWorker.Shutdown(shutdownContext); err != nil {
			logger.Error("background worker shutdown failed", "error", err)
		}
		if err := server.Shutdown(shutdownContext); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	case err := <-serverError:
		logger.Error("HTTP server stopped", "error", err)
		os.Exit(1)
	}
}

func authRateLimit(next http.Handler, limiter ports.RateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" || r.URL.Path == "/register" || r.URL.Path == "/forgot-password" || r.URL.Path == "/reset-password" || r.URL.Path == "/verify-email" || strings.HasPrefix(r.URL.Path, "/api/v1/auth/") {
			allowed, err := limiter.Allow(r.Context(), "auth:"+clientAddress(r), 20, time.Minute)
			if err != nil {
				writeRateLimitAPIError(w, r, http.StatusServiceUnavailable, "RATE_LIMIT_UNAVAILABLE", "Authentication is temporarily unavailable.")
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", "60")
				writeRateLimitAPIError(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "Too many authentication attempts.")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func clientAddress(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func writeRateLimitAPIError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		http.Error(w, message, status)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{
		"code": code, "message": message, "request_id": w.Header().Get("X-Request-ID"),
	}})
}

func tcpProbe(address string) healthTransport.Probe {
	return func(ctx context.Context) error {
		dialer := net.Dialer{Timeout: 2 * time.Second}
		connection, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return err
		}
		return connection.Close()
	}
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = time.Now().UTC().Format("20060102T150405.000000000Z07:00")
		}
		w.Header().Set("X-Request-ID", requestID)
		r = r.WithContext(ports.WithRequestID(r.Context(), requestID))
		next.ServeHTTP(w, r)
	})
}
