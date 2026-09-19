package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wecratfs/commerce/internal/apphost"
	"github.com/wecratfs/commerce/internal/config"
	"github.com/wecratfs/commerce/internal/worker"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	cfg := config.FromEnv()

	startupContext, startupCancel := context.WithTimeout(context.Background(), 20*time.Second)
	runtime, err := apphost.New(startupContext, cfg, logger)
	startupCancel()
	if err != nil {
		logger.Error("initialize application", "error", err)
		os.Exit(1)
	}
	defer runtime.Close()

	server := &http.Server{
		Addr: cfg.Address, Handler: runtime.Handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverError := make(chan error, 1)
	go func() {
		logger.Info("WeeVCrafts server listening", "address", cfg.Address)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverError <- err
		}
	}()

	maintenanceContext, stopMaintenance := context.WithCancel(context.Background())
	defer stopMaintenance()
	go worker.RunPeriodic(maintenanceContext, time.Minute, func(ctx context.Context) error {
		jobContext, cancel := context.WithTimeout(ctx, 25*time.Second)
		defer cancel()
		return runtime.RunMaintenance(jobContext)
	})

	shutdownSignal, stopSignal := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignal()
	select {
	case <-shutdownSignal.Done():
		stopMaintenance()
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	case err := <-serverError:
		logger.Error("HTTP server stopped", "error", err)
		os.Exit(1)
	}
}
