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

	httpdelivery "order-service/internal/delivery/http"
	"order-service/internal/repository"
	"order-service/internal/usecase"
	"order-service/pkg/paymentclient"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	dbPath := getenv("ORDER_DB", "./order.db")
	paymentAddr := getenv("PAYMENT_GRPC_ADDR", "localhost:50051")

	repo, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		logger.Error("failed to init sqlite", "err", err)
		os.Exit(1)
	}
	defer func() {
		_ = repo.Close()
	}()

	payCli, err := paymentclient.New(paymentAddr)
	if err != nil {
		logger.Error("failed to create payment grpc client", "err", err)
		os.Exit(1)
	}
	defer func() {
		_ = payCli.Close()
	}()

	uc := usecase.NewOrderUseCase(repo, payCli, logger)
	h := httpdelivery.NewHandler(uc)
	router := httpdelivery.NewRouter(h, repo, logger)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("order-service starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	} else {
		logger.Info("server stopped")
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
