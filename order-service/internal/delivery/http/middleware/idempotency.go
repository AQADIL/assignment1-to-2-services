package middleware

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"order-service/internal/domain"
)

type bodyCaptureWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *bodyCaptureWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	_, _ = w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func Idempotency(repo domain.IdempotencyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost || c.Request.URL == nil || c.Request.URL.Path != "/orders" {
			c.Next()
			return
		}

		key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if key == "" {
			c.Next()
			return
		}

		statusCode, body, found, err := repo.Get(c.Request.Context(), key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to check idempotency key"})
			return
		}
		if found {
			c.Status(statusCode)
			c.Writer.Header().Set("Content-Type", "application/json")
			_, _ = io.Copy(c.Writer, bytes.NewReader(body))
			c.Abort()
			return
		}

		bcw := &bodyCaptureWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = bcw
		c.Next()

		if bcw.status == 0 {
			bcw.status = c.Writer.Status()
		}
		if bcw.body.Len() == 0 {
			return
		}
		if err := repo.Save(c.Request.Context(), key, bcw.status, bcw.body.Bytes()); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
		}
	}
}
