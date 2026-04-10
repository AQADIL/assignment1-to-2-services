package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/AQADIL/assignment2-generated/order"
	grpclib "google.golang.org/grpc"

	grpcdelivery "order-service/internal/delivery/grpc"
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

	// --- HTTP server (REST, kept for backward compat) ---
	h := httpdelivery.NewHandler(uc)
	router := httpdelivery.NewRouter(h, repo, logger)
	httpPort := getenv("HTTP_PORT", "8080")

	srv := &http.Server{
		Addr:              ":" + httpPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("order-service HTTP starting", "port", httpPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	// --- gRPC server (streaming + all order RPCs) ---
	grpcPort := getenv("GRPC_PORT", "50052")
	grpcSrv := grpclib.NewServer()
	pb.RegisterOrderServiceServer(grpcSrv, grpcdelivery.NewServer(uc, repo))

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		logger.Error("failed to listen grpc", "err", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("order-service gRPC starting", "port", grpcPort)
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Error("grpc server error", "err", err)
			os.Exit(1)
		}
	}()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown signal received")
	grpcSrv.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("http graceful shutdown failed", "err", err)
	}
	logger.Info("server stopped")
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
