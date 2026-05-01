package nats

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
)

type Publisher struct {
	js      nats.JetStreamContext
	subject string
}

type PaymentCompletedEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

func NewPublisher(js nats.JetStreamContext, subject string) *Publisher {
	return &Publisher{js: js, subject: subject}
}

func (p *Publisher) EnsureStream(ctx context.Context) error {
	_, err := p.js.AddStream(&nats.StreamConfig{
		Name:     "PAYMENTS",
		Subjects: []string{"payments.completed", "payments.completed.dlq"},
		Storage:  nats.FileStorage,
	})
	if err != nil {
		return err
	}
	return nil
}

func (p *Publisher) PublishPaymentCompleted(ctx context.Context, ev PaymentCompletedEvent) error {
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = p.js.Publish(p.subject, b, nats.Context(ctx))
	return err
}
