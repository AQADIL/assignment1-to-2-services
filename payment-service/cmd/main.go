package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/AQADIL/assignment2-generated/payment"
	"github.com/nats-io/nats.go"
	grpclib "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	grpcdelivery "payment-service/internal/delivery/grpc"
	natsdelivery "payment-service/internal/delivery/nats"
	"payment-service/internal/repository"
	"payment-service/internal/usecase"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	dbPath := getenv("PAYMENT_DB", "./payment.db")
	grpcPort := getenv("GRPC_PORT", "50051")
	natsURL := getenv("NATS_URL", "nats://localhost:4222")

	repo, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		logger.Error("failed to init sqlite", "err", err)
		os.Exit(1)
	}
	defer func() {
		_ = repo.Close()
	}()

	nc, err := nats.Connect(natsURL)
	if err != nil {
		logger.Error("failed to connect nats", "err", err)
		os.Exit(1)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		logger.Error("failed to init jetstream", "err", err)
		os.Exit(1)
	}

	publisher := natsdelivery.NewPublisher(js, "payments.completed")
	if err := publisher.EnsureStream(context.Background()); err != nil {
		logger.Error("failed to ensure stream", "err", err)
		os.Exit(1)
	}

	uc := usecase.NewPaymentUseCaseWithPublisher(repo, publisher)

	grpcServer := grpclib.NewServer(
		grpclib.UnaryInterceptor(grpcdelivery.LoggingInterceptor(logger)),
	)
	pb.RegisterPaymentServiceServer(grpcServer, grpcdelivery.NewServer(uc))
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		logger.Error("failed to listen", "err", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("payment-service gRPC starting", "port", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("grpc server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown signal received")
	grpcServer.GracefulStop()
	logger.Info("server stopped")
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
