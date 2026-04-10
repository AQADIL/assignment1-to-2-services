package grpc

import (
	"context"
	"log/slog"
	"time"

	grpclib "google.golang.org/grpc"
)

func LoggingInterceptor(logger *slog.Logger) grpclib.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpclib.UnaryServerInfo,
		handler grpclib.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		dur := time.Since(start)
		logger.Info("grpc_request",
			"method", info.FullMethod,
			"duration_ms", dur.Milliseconds(),
			"error", err,
		)

		return resp, err
	}
}
