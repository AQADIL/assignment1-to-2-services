package usecase

import (
	"context"
	natsdelivery "payment-service/internal/delivery/nats"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentUseCase struct {
	repo      domain.PaymentRepository
	publisher *natsdelivery.Publisher
}

func NewPaymentUseCase(repo domain.PaymentRepository) *PaymentUseCase {
	return &PaymentUseCase{repo: repo}
}

func NewPaymentUseCaseWithPublisher(repo domain.PaymentRepository, publisher *natsdelivery.Publisher) *PaymentUseCase {
	return &PaymentUseCase{repo: repo, publisher: publisher}
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

	if uc.publisher != nil {
		// DLQ simulation: if order_id starts with "dlq-", set Fail=true
		failDLQ := len(p.OrderID) > 4 && p.OrderID[:4] == "dlq-"
		_ = uc.publisher.PublishPaymentCompleted(ctx, natsdelivery.PaymentCompletedEvent{
			EventID:       uuid.NewString(),
			OrderID:       p.OrderID,
			Amount:        p.Amount,
			CustomerEmail: "ТЕНТЕК@gmail.com",
			Status:        p.Status,
			Fail:          failDLQ,
		})
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
