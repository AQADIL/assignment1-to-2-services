package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notification-service/internal/repository"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	dbPath := getenv("NOTIFICATION_DB", "./notification.db")

	repo, err := repository.NewSQLiteRepository(dbPath)
	if err != nil {
		logger.Error("failed to init sqlite", "err", err)
		os.Exit(1)
	}
	defer func() {
		_ = repo.Close()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := repo.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		logger.Error("failed to close db", "err", err)
	}

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
