package usecase

import (
	"context"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentUseCase struct {
	repo domain.PaymentRepository
}

func NewPaymentUseCase(repo domain.PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repo: repo}
}

func (uc *PaymentUseCase) CreatePayment(ctx context.Context, req domain.CreatePaymentRequest) (domain.CreatePaymentResponse, error) {
	status := domain.PaymentStatusAuthorized
	if req.Amount > 100000 {
		status = domain.PaymentStatusDeclined
	}

	p := domain.Payment{
		ID:            uuid.NewString(),
		OrderID:       req.OrderID,
		TransactionID: uuid.NewString(),
		Amount:        req.Amount,
		Status:        status,
	}

	if err := uc.repo.Create(ctx, p); err != nil {
		return domain.CreatePaymentResponse{}, err
	}

	return domain.CreatePaymentResponse{
		PaymentID:     p.ID,
		TransactionID: p.TransactionID,
		Status:        p.Status,
	}, nil
}

func (uc *PaymentUseCase) GetPayment(ctx context.Context, orderID string) (domain.Payment, error) {
	return uc.repo.GetByOrderID(ctx, orderID)
}

func (uc *PaymentUseCase) ListPayments(ctx context.Context, minAmount, maxAmount int64) ([]domain.Payment, error) {
	if minAmount > 0 && maxAmount > 0 && minAmount > maxAmount {
		return nil, domain.ErrInvalidRange
	}
	return uc.repo.FindByAmountRange(ctx, minAmount, maxAmount)
}
