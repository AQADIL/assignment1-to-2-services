package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"

	"notification-service/internal/email"
)

type ProcessedStore interface {
	IsProcessed(ctx context.Context, eventID string) (bool, error)
	MarkProcessed(ctx context.Context, eventID string) error
}

type PaymentCompletedEvent struct {
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
	Fail          bool   `json:"fail,omitempty"`
}

type Consumer struct {
	js         nats.JetStreamContext
	sub        *nats.Subscription
	store      ProcessedStore
	sender     email.Sender
	dlqSubject string
}

func NewConsumer(js nats.JetStreamContext, store ProcessedStore, sender email.Sender, dlqSubject string) *Consumer {
	return &Consumer{js: js, store: store, sender: sender, dlqSubject: dlqSubject}
}

func (c *Consumer) Start(ctx context.Context, subject, durable string) error {
	sub, err := c.js.PullSubscribe(subject, durable, nats.ManualAck())
	if err != nil {
		return err
	}
	c.sub = sub

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msgs, err := sub.Fetch(1, nats.Context(ctx))
			if err != nil {
				continue
			}

			for _, msg := range msgs {
				_ = c.handle(ctx, msg)
			}
		}
	}()

	return nil
}

func (c *Consumer) Stop() error {
	if c.sub != nil {
		return c.sub.Unsubscribe()
	}
	return nil
}

func (c *Consumer) handle(ctx context.Context, msg *nats.Msg) error {
	var ev PaymentCompletedEvent
	if err := json.Unmarshal(msg.Data, &ev); err != nil {
		_ = c.publishDLQ(ctx, msg.Data)
		_ = msg.Ack()
		return err
	}
	if ev.Fail {
		_ = c.publishDLQ(ctx, msg.Data)
		_ = msg.Ack()
		return fmt.Errorf("simulated failure")
	}

	processed, err := c.store.IsProcessed(ctx, ev.EventID)
	if err != nil {
		return err
	}
	if processed {
		_ = msg.Ack()
		return nil
	}

	if err := c.sender.Send(ctx, email.Message{
		To:      ev.CustomerEmail,
		OrderID: ev.OrderID,
		Amount:  ev.Amount,
	}); err != nil {
		return fmt.Errorf("email send failed: %w", err)
	}

	if err := c.store.MarkProcessed(ctx, ev.EventID); err != nil {
		return err
	}

	_ = msg.Ack()
	return nil
}

func (c *Consumer) publishDLQ(ctx context.Context, payload []byte) error {
	if c.dlqSubject == "" {
		return nil
	}
	_, err := c.js.Publish(c.dlqSubject, payload, nats.Context(ctx))
	return err
}
