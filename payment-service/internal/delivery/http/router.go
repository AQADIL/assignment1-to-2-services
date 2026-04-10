package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func NewRouter(h *Handler, logger *slog.Logger) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger(logger))

	r.POST("/payments", h.CreatePayment)
	r.GET("/payments/:order_id", h.GetPayment)

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
