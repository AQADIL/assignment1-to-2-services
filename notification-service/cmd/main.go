package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	natsdelivery "notification-service/internal/delivery/nats"
	"notification-service/internal/email"
	"notification-service/internal/repository"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	dbPath := getenv("NOTIFICATION_DB", "./notification.db")
	natsURL := getenv("NATS_URL", "nats://localhost:4222")
	redisURL := getenv("REDIS_URL", "redis://localhost:6379")

	sqliteRepo, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		logger.Error("failed to init sqlite", "err", err)
		os.Exit(1)
	}
	defer func() {
		_ = sqliteRepo.Close()
	}()

	redisStore, err := repository.NewRedisStore(redisURL, 24*time.Hour)
	if err != nil {
		logger.Warn("redis unavailable, falling back to sqlite for idempotency", "err", err)
	}

	var store natsdelivery.ProcessedStore
	if redisStore != nil {
		store = redisStore
		defer func() { _ = redisStore.Close() }()
		logger.Info("idempotency store: redis")
	} else {
		store = sqliteRepo
		logger.Info("idempotency store: sqlite")
	}

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

	if _, err := js.AddStream(&nats.StreamConfig{
		Name:     "PAYMENTS",
		Subjects: []string{"payments.completed", "payments.completed.dlq"},
		Storage:  nats.FileStorage,
	}); err != nil {
		logger.Error("failed to ensure stream", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var sender email.Sender
	switch getenv("PROVIDER_MODE", "SIMULATED") {
	case "SMTP":
		sender = email.NewSMTPProvider(email.SMTPConfig{
			Host:     getenv("SMTP_HOST", "localhost"),
			Port:     getenv("SMTP_PORT", "25"),
			Username: getenv("SMTP_USER", ""),
			Password: getenv("SMTP_PASS", ""),
			From:     getenv("SMTP_FROM", "noreply@example.com"),
		})
		logger.Info("email provider: SMTP")
	default:
		sender = email.NewMockProvider()
		logger.Info("email provider: SIMULATED (MockProvider)")
	}

	protectedSender := email.NewCircuitBreaker(sender, 3, 30*time.Second)

	logger.Info("notification-service starting", "nats_url", natsURL, "db_path", dbPath)

	consumer := natsdelivery.NewConsumer(js, store, protectedSender, "payments.completed.dlq")
	if err := consumer.Start(ctx, "payments.completed", "notification"); err != nil {
		logger.Error("failed to start consumer", "err", err)
		os.Exit(1)
	}
	logger.Info("notification-service consumer started")

	<-ctx.Done()
	_ = consumer.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	<-shutdownCtx.Done()
	logger.Info("notification-service stopped")
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
