// Package apphost composes the persistent application for both the VPS HTTP
// server and Netlify's request/scheduled functions.
package apphost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	redisclient "github.com/redis/go-redis/v9"

	emailadapter "github.com/wecratfs/commerce/internal/adapters/email"
	"github.com/wecratfs/commerce/internal/adapters/meilisearch"
	metricsadapter "github.com/wecratfs/commerce/internal/adapters/metrics"
	"github.com/wecratfs/commerce/internal/adapters/postgres"
	redisadapter "github.com/wecratfs/commerce/internal/adapters/redis"
	storageadapter "github.com/wecratfs/commerce/internal/adapters/storage"
	applicationauth "github.com/wecratfs/commerce/internal/application/auth"
	applicationcatalog "github.com/wecratfs/commerce/internal/application/catalog"
	applicationcommerce "github.com/wecratfs/commerce/internal/application/commerce"
	applicationprivacy "github.com/wecratfs/commerce/internal/application/privacy"
	"github.com/wecratfs/commerce/internal/config"
	"github.com/wecratfs/commerce/internal/ports"
	apiTransport "github.com/wecratfs/commerce/internal/transport/api"
	healthTransport "github.com/wecratfs/commerce/internal/transport/health"
	webTransport "github.com/wecratfs/commerce/internal/transport/web"
)

type Runtime struct {
	Handler http.Handler
	pool    *pgxpool.Pool
	redis   *redisclient.Client
	netlify bool

	logger                *slog.Logger
	productIndexProcessor *applicationcatalog.ProductIndexProcessor
	commerceRepository    *postgres.CommerceRepository
	mediaService          *applicationcommerce.MediaService
}

func NewNetlify(logger *slog.Logger) (*Runtime, error) {
	cfg := config.FromNetlifyEnv()
	if err := cfg.ValidateNetlify(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return New(ctx, cfg, logger)
}

// New builds a request handler without applying schema changes or seeding
// business data. Migrations are run explicitly by cmd/migrate.
func New(ctx context.Context, cfg config.Config, logger *slog.Logger) (_ *Runtime, resultErr error) {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.DatabaseURL == "" {
		return nil, errors.New("database configuration is required")
	}
	if !cfg.NetlifyRuntime && (cfg.RedisAddress == "" || cfg.S3Address == "" || cfg.S3Bucket == "") {
		return nil, errors.New("database, Redis, and object storage configuration are required")
	}
	if cfg.DatabaseMaxConns < 1 || cfg.DatabaseMaxConns > 20 {
		return nil, errors.New("database pool size must be between 1 and 20")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL configuration: %w", err)
	}
	poolConfig.MaxConns = cfg.DatabaseMaxConns
	poolConfig.MinConns = 0
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}
	runtime := &Runtime{pool: pool, logger: logger, netlify: cfg.NetlifyRuntime}
	defer func() {
		if resultErr != nil {
			runtime.Close()
		}
	}()
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	var rateLimiter ports.RateLimiter
	var searchCache interface {
		ports.Cache
		ports.CacheInvalidator
	}
	var sessionStore ports.SessionStore
	var mediaStorage ports.ObjectStorage
	if cfg.NetlifyRuntime {
		rateLimiter = postgres.NewRateLimiter(pool)
		searchCache = postgres.NewCache(pool)
		sessionStore = postgres.NewSessionStore(pool)
		mediaStorage = postgres.NewDatabaseMediaStorage(pool)
	} else {
		redisOptions, err := redisOptions(cfg.RedisAddress)
		if err != nil {
			return nil, fmt.Errorf("parse Redis configuration: %w", err)
		}
		runtime.redis = redisclient.NewClient(redisOptions)
		if err := runtime.redis.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("connect to Redis: %w", err)
		}
		rateLimiter = redisadapter.NewRateLimiter(runtime.redis)
		searchCache = redisadapter.NewCache(runtime.redis)
		sessionStore = redisadapter.NewSessionStore(runtime.redis)
		s3Storage, err := storageadapter.NewS3(cfg.S3Address, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3Secure)
		if err != nil {
			return nil, fmt.Errorf("configure S3-compatible media storage: %w", err)
		}
		if cfg.S3AutoCreateBucket {
			if err := ensureBucket(ctx, s3Storage); err != nil {
				return nil, fmt.Errorf("ensure media bucket: %w", err)
			}
		} else if err := s3Storage.VerifyBucket(ctx); err != nil {
			return nil, fmt.Errorf("verify configured media bucket: %w", err)
		}
		mediaStorage = s3Storage
	}

	catalogService := applicationcatalog.NewService(postgres.NewProductRepository(pool))
	searchRepository := postgres.NewProductSearchRepository(pool)
	searchMetrics := metricsadapter.NewSearchMetrics()
	var searchEngine ports.SearchEngine
	var searchIndexer ports.SearchIndexer
	if strings.TrimSpace(cfg.MeiliAddress) != "" {
		engine := meilisearch.NewSearchEngineWithKey(cfg.MeiliAddress, cfg.MeiliAPIKey)
		searchEngine, searchIndexer = engine, engine
	}
	searchService := applicationcatalog.NewSearchServiceWithEngineAndObserver(searchRepository, searchCache, searchEngine, searchMetrics)
	passwordHasher := applicationauth.NewArgon2idHasher()
	userRepository := postgres.NewUserRepository(pool)
	authService := applicationauth.NewService(userRepository, passwordHasher)
	sessionService := applicationauth.NewSessionService(sessionStore)
	var emailSender ports.EmailSender
	if cfg.SMTPAddress != "" && cfg.SMTPFrom != "" {
		emailSender = emailadapter.NewSMTP(cfg.SMTPAddress, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
	}
	recoveryService := applicationauth.NewRecoveryService(postgres.NewAuthActionRepository(pool), passwordHasher, emailSender, cfg.PublicURL)
	commerceRepository := postgres.NewCommerceRepository(pool)
	runtime.commerceRepository = commerceRepository
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
	customerDirectoryService := applicationauth.NewCustomerDirectoryService(userRepository, roleAuthorizer)
	financeService := applicationcommerce.NewFinanceService(commerceRepository, roleAuthorizer)
	supportService := applicationcommerce.NewSupportService(commerceRepository, roleAuthorizer)
	returnService := applicationcommerce.NewReturnService(commerceRepository, roleAuthorizer)
	cartService := applicationcommerce.NewCartService(commerceRepository)
	orderService := applicationcommerce.NewOrderService(commerceRepository)
	privacyService := applicationprivacy.NewService(postgres.NewPrivacyRepository(pool))
	runtime.mediaService = applicationcommerce.NewMediaService(mediaStorage, postgres.NewMediaRepository(pool))

	webHandler, err := webTransport.NewFullHandlerWithPaymentConfigMediaRefundAndRecovery(
		catalogService, authService, sessionService, commerceCatalog, cartService,
		orderService, nil, searchService, privacyService, runtime.mediaService,
		nil, recoveryService, cfg.SecureCookies, "",
	)
	if err != nil {
		return nil, fmt.Errorf("load web templates: %w", err)
	}
	webHandler.SetManualCheckout(true)
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

	apiHandler := apiTransport.NewCompleteHandlerWithMediaPaymentRefundAndRecovery(
		catalogService, searchService, authService, sessionService, cartService,
		orderService, privacyService, runtime.mediaService, nil, nil, recoveryService,
		cfg.SecureCookies,
	)
	apiHandler.SetManualCheckout(true)

	if searchIndexer != nil {
		runtime.productIndexProcessor = applicationcatalog.NewProductIndexProcessorWithCache(
			postgres.NewOutboxRepository(pool), searchRepository, searchIndexer, searchCache,
		)
	}

	healthProbes := map[string]healthTransport.Probe{
		"postgresql": func(ctx context.Context) error { return postgres.Ping(ctx, pool) },
	}
	if runtime.redis != nil {
		healthProbes["redis"] = func(ctx context.Context) error { return runtime.redis.Ping(ctx).Err() }
		healthProbes["s3"] = tcpProbe(cfg.S3Address)
	}
	if cfg.MeiliAddress != "" {
		healthProbes["meilisearch"] = tcpProbe(cfg.MeiliAddress)
	}
	healthHandler := healthTransport.NewHandlerWithMetrics(healthProbes, func() any { return searchMetrics.Snapshot() })
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
	mux.HandleFunc("/api/v1/seller/products/media-upload", apiHandler.MediaUpload)
	mux.HandleFunc("/api/v1/seller/products/media-finalize", apiHandler.MediaFinalize)
	mux.HandleFunc("/api/v1/seller/products/media-delete", apiHandler.MediaDelete)
	mux.Handle("/", webHandler)
	runtime.Handler = requestID(authRateLimit(routeAliases(mux), rateLimiter))
	return runtime, nil
}

func ensureBucket(ctx context.Context, storage *storageadapter.S3) error {
	deadline := time.NewTimer(8 * time.Second)
	defer deadline.Stop()
	for {
		if err := storage.EnsureBucket(ctx); err == nil {
			return nil
		}
		wait := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			wait.Stop()
			return ctx.Err()
		case <-deadline.C:
			wait.Stop()
			return errors.New("media bucket could not be created before startup deadline")
		case <-wait.C:
		}
	}
}

func (r *Runtime) Close() {
	if r.redis != nil {
		_ = r.redis.Close()
	}
	if r.pool != nil {
		r.pool.Close()
	}
}

// RunMaintenance performs one bounded, idempotent batch for a scheduled
// function or the VPS scheduler. It never starts a resident worker.
func (r *Runtime) RunMaintenance(ctx context.Context) error {
	var failures []error
	if r.productIndexProcessor != nil {
		indexContext, cancel := context.WithTimeout(ctx, 8*time.Second)
		_, err := r.productIndexProcessor.Process(indexContext, 1)
		cancel()
		if err != nil {
			failures = append(failures, fmt.Errorf("process product index outbox: %w", err))
		}
	}
	if r.netlify {
		stateContext, cancelState := context.WithTimeout(ctx, 3*time.Second)
		if err := postgres.CleanupRuntimeState(stateContext, r.pool, 1000); err != nil {
			failures = append(failures, fmt.Errorf("clean expired runtime state: %w", err))
		}
		cancelState()
	}
	reservationContext, cancelReservations := context.WithTimeout(ctx, 4*time.Second)
	_, err := r.commerceRepository.ReleaseExpiredReservations(reservationContext, 100)
	cancelReservations()
	if err != nil {
		failures = append(failures, fmt.Errorf("release expired inventory reservations: %w", err))
	}
	mediaContext, cancelMedia := context.WithTimeout(ctx, 10*time.Second)
	_, err = r.mediaService.CleanupExpired(mediaContext, 20)
	cancelMedia()
	if err != nil && !errors.Is(err, applicationcommerce.ErrInvalidMedia) {
		failures = append(failures, fmt.Errorf("clean expired product media: %w", err))
	}
	if err := errors.Join(failures...); err != nil {
		r.logger.Error("scheduled maintenance batch failed", "error", err)
		return err
	}
	r.logger.Info("scheduled maintenance batch completed")
	return nil
}

func redisOptions(address string) (*redisclient.Options, error) {
	if strings.Contains(address, "://") {
		return redisclient.ParseURL(address)
	}
	return &redisclient.Options{Addr: address}, nil
}

func tcpProbe(address string) healthTransport.Probe {
	target := address
	if parsed, err := url.Parse(address); err == nil && parsed.Host != "" {
		target = parsed.Host
	}
	return func(ctx context.Context) error {
		dialer := net.Dialer{Timeout: 2 * time.Second}
		connection, err := dialer.DialContext(ctx, "tcp", target)
		if err != nil {
			return err
		}
		return connection.Close()
	}
}

func routeAliases(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case path == "/seller-admin" || strings.HasPrefix(path, "/seller-admin/"):
			r.URL.Path = "/seller" + strings.TrimPrefix(path, "/seller-admin")
		case path == "/support-portal" || strings.HasPrefix(path, "/support-portal/"):
			r.URL.Path = "/support" + strings.TrimPrefix(path, "/support-portal")
		case path == "/admin":
			r.URL.Path = "/admin/sellers"
		case path == "/account":
			r.URL.Path = "/account/sessions"
		}
		next.ServeHTTP(w, r)
	})
}

func authRateLimit(next http.Handler, limiter ports.RateLimiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && (r.URL.Path == "/login" || r.URL.Path == "/register" || r.URL.Path == "/forgot-password" || r.URL.Path == "/reset-password" || r.URL.Path == "/verify-email" || strings.HasPrefix(r.URL.Path, "/api/v1/auth/")) {
			allowed, err := limiter.Allow(r.Context(), "auth:"+clientAddress(r), 40, time.Minute)
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

func AuthRateLimit(next http.Handler, limiter ports.RateLimiter) http.Handler {
	return authRateLimit(next, limiter)
}

func clientAddress(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
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

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" || len(id) > 128 || strings.ContainsAny(id, "\r\n") {
			id = time.Now().UTC().Format("20060102T150405.000000000Z07:00")
		}
		w.Header().Set("X-Request-ID", id)
		r = r.WithContext(ports.WithRequestID(r.Context(), id))
		next.ServeHTTP(w, r)
	})
}
