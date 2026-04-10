package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"order-service/internal/delivery/http/middleware"
	"order-service/internal/domain"
)

func NewRouter(h *Handler, idemRepo domain.IdempotencyRepository, logger *slog.Logger) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger(logger))
	r.Use(middleware.Idempotency(idemRepo))

	r.GET("/orders", h.ListOrders)
	r.POST("/orders", h.CreateOrder)
	r.GET("/orders/:id", h.GetOrder)
	r.PATCH("/orders/:id/cancel", h.CancelOrder)

	return r
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		dur := time.Since(start)
		status := c.Writer.Status()
		logger.Info("http_request",
			"method", method,
			"path", path,
			"status", status,
			"duration_ms", dur.Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}
