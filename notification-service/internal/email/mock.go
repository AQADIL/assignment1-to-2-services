package email

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type MockProvider struct {
	rng *rand.Rand
}

func NewMockProvider() *MockProvider {
	return &MockProvider{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

func (p *MockProvider) Send(ctx context.Context, msg Message) error {
	time.Sleep(time.Duration(50+p.rng.Intn(150)) * time.Millisecond)

	if p.rng.Float64() < 0.30 {
		return errors.New("mock provider: transient network error")
	}

	fmt.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%d\n", msg.To, msg.OrderID, msg.Amount)
	return nil
}
