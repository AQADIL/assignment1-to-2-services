package domain

import (
	"context"
	"errors"
)

const (
	PaymentStatusAuthorized = "Authorized"
	PaymentStatusDeclined   = "Declined"
)

var (
	ErrPaymentNotFound = errors.New("payment not found")
)

type Payment struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
}

type CreatePaymentRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Amount  int64  `json:"amount" binding:"required,gt=0"`
}

type CreatePaymentResponse struct {
	PaymentID     string `json:"payment_id"`
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

type PaymentRepository interface {
	Create(ctx context.Context, p Payment) error
	GetByOrderID(ctx context.Context, orderID string) (Payment, error)
}

type PaymentUseCase interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResponse, error)
	GetPayment(ctx context.Context, orderID string) (Payment, error)
}
