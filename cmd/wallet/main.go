// Package main ...
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"wallet/internal/driver/sqlstore"
	"wallet/internal/port"
	"wallet/internal/port/handler"
	"wallet/internal/port/middleware"
	"wallet/internal/repository"
	"wallet/internal/usecase"
	"wallet/pkg/logger"
)

func main() {
	// --- Config ---
	cfg, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	// --- Logger ---
	log := logger.NewLogger(cfg.LogLevel).With(
		slog.String("bind_addr", cfg.BindAddr),
	)

	log.Info("starting service")

	// --- Database ---
	ctx := context.Background()

	pcfg := sqlstore.DefaultPoolConfig()

	store, err := sqlstore.New(ctx, cfg.DatabaseURL, pcfg)
	if err != nil {
		log.Error("failed to connect to database", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer store.Close()

	log.Info("database connected")

	repo := repository.New(store.Pool())
	uc := usecase.New(repo, store)

	serverAPI := port.NewServer(log)
	walletHandler := handler.NewWalletHandler(uc, serverAPI)

	mux := http.NewServeMux()

	mux.Handle("POST /api/v1/wallet", walletHandler.HandleOperation())
	mux.Handle("GET /api/v1/wallets/{id}", walletHandler.HandleGetBalance())

	middleware.Use(middleware.RequestID)
	httpHandler := middleware.Apply(mux)

	srv := &http.Server{
		Addr:         cfg.BindAddr,
		Handler:      httpHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", slog.String("addr", cfg.BindAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Error("server error", slog.String("err", err.Error()))
		os.Exit(1)
	case sig := <-quit:
		log.Info("shutdown signal received", slog.String("signal", sig.String()))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", slog.String("err", err.Error()))
		os.Exit(1)
	}

	log.Info("service stopped")
}
