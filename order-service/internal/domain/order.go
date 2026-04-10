package domain

import (
	"context"
	"time"
)

const (
	OrderStatusPending   = "Pending"
	OrderStatusPaid      = "Paid"
	OrderStatusFailed    = "Failed"
	OrderStatusCancelled = "Cancelled"
)

type Order struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customer_id"`
	ItemName   string    `json:"item_name"`
	Amount     int64     `json:"amount"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type CreateOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name" binding:"required"`
	Amount     int64  `json:"amount" binding:"required,gt=0"`
}

type PaymentCreateRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Amount  int64  `json:"amount" binding:"required,gt=0"`
}

type PaymentCreateResponse struct {
	PaymentID     string `json:"payment_id"`
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

type CancelOrderResponse struct {
	Order Order `json:"order"`
}

type OrderRepository interface {
	Create(ctx context.Context, order Order) error
	GetByID(ctx context.Context, id string) (Order, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	ListByCustomerID(ctx context.Context, customerID string) ([]Order, error)
	Subscribe(customerID string) (<-chan Order, string)
	Unsubscribe(subID string)
}

type IdempotencyRepository interface {
	Get(ctx context.Context, key string) (statusCode int, body []byte, found bool, err error)
	Save(ctx context.Context, key string, statusCode int, body []byte) error
}

type PaymentClient interface {
	CreatePayment(ctx context.Context, req PaymentCreateRequest) (PaymentCreateResponse, error)
}

type OrderUseCase interface {
	CreateOrder(ctx context.Context, req CreateOrderRequest) (Order, error)
	GetOrder(ctx context.Context, id string) (Order, error)
	CancelOrder(ctx context.Context, id string) (Order, error)
	ListOrdersByCustomer(ctx context.Context, customerID string) ([]Order, error)
}
