package usecase

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"order-service/internal/domain"
)

const orderCacheTTL = 5 * time.Minute

type OrderUseCase struct {
	repo   domain.OrderRepository
	pay    domain.PaymentClient
	cache  domain.OrderCache
	logger *slog.Logger
}

func NewOrderUseCase(repo domain.OrderRepository, pay domain.PaymentClient, logger *slog.Logger) *OrderUseCase {
	return &OrderUseCase{repo: repo, pay: pay, logger: logger}
}

func NewOrderUseCaseWithCache(repo domain.OrderRepository, pay domain.PaymentClient, cache domain.OrderCache, logger *slog.Logger) *OrderUseCase {
	return &OrderUseCase{repo: repo, pay: pay, cache: cache, logger: logger}
}

func (uc *OrderUseCase) CreateOrder(ctx context.Context, req domain.CreateOrderRequest) (domain.Order, error) {
	if req.Amount <= 0 {
		return domain.Order{}, domain.ErrInvalidAmount
	}

	order := domain.Order{
		ID:         uuid.NewString(),
		CustomerID: req.CustomerID,
		ItemName:   req.ItemName,
		Amount:     req.Amount,
		Status:     domain.OrderStatusPending,
		CreatedAt:  time.Now().UTC(),
	}

	if err := uc.repo.Create(ctx, order); err != nil {
		return domain.Order{}, err
	}

	payResp, err := uc.pay.CreatePayment(ctx, domain.PaymentCreateRequest{OrderID: order.ID, Amount: order.Amount})
	if err != nil {
		uc.logger.Error("payment call failed", "err", err, "order_id", order.ID)
		_ = uc.repo.UpdateStatus(ctx, order.ID, domain.OrderStatusFailed)
		return domain.Order{}, errors.Join(domain.ErrPaymentUnavailable, err)
	}

	if payResp.Status == "Authorized" {
		if err := uc.repo.UpdateStatus(ctx, order.ID, domain.OrderStatusPaid); err != nil {
			return domain.Order{}, err
		}
	} else {
		if err := uc.repo.UpdateStatus(ctx, order.ID, domain.OrderStatusFailed); err != nil {
			return domain.Order{}, err
		}
	}

	updated, err := uc.repo.GetByID(ctx, order.ID)
	if err != nil {
		return domain.Order{}, err
	}
	if uc.cache != nil {
		_ = uc.cache.Set(ctx, updated, orderCacheTTL)
	}
	return updated, nil
}

func (uc *OrderUseCase) GetOrder(ctx context.Context, id string) (domain.Order, error) {
	if uc.cache != nil {
		if cached, ok, err := uc.cache.Get(ctx, id); err == nil && ok {
			return cached, nil
		}
	}
	order, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if uc.cache != nil {
		_ = uc.cache.Set(ctx, order, orderCacheTTL)
	}
	return order, nil
}

func (uc *OrderUseCase) CancelOrder(ctx context.Context, id string) (domain.Order, error) {
	o, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if o.Status != domain.OrderStatusPending {
		return domain.Order{}, domain.ErrCannotCancelOrder
	}
	if err := uc.repo.UpdateStatus(ctx, id, domain.OrderStatusCancelled); err != nil {
		return domain.Order{}, err
	}
	if uc.cache != nil {
		_ = uc.cache.Delete(ctx, id)
	}
	updated, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}
	if uc.cache != nil {
		_ = uc.cache.Set(ctx, updated, orderCacheTTL)
	}
	return updated, nil
}

func (uc *OrderUseCase) ListOrdersByCustomer(ctx context.Context, customerID string) ([]domain.Order, error) {
	return uc.repo.ListByCustomerID(ctx, customerID)
}
