package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"payment-service/internal/domain"
)

type Handler struct {
	uc domain.PaymentUseCase
}

func NewHandler(uc domain.PaymentUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) CreatePayment(c *gin.Context) {
	var req domain.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	resp, err := h.uc.CreatePayment(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) GetPayment(c *gin.Context) {
	orderID := c.Param("order_id")
	pay, err := h.uc.GetPayment(c.Request.Context(), orderID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, pay)
}
