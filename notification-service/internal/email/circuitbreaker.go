package email

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type cbState int

const (
	stateClosed   cbState = iota
	stateOpen     cbState = iota
	stateHalfOpen cbState = iota
)

type CircuitBreaker struct {
	mu           sync.Mutex
	inner        Sender
	maxFailures  int
	openDuration time.Duration
	failures     int
	state        cbState
	openedAt     time.Time
}

func NewCircuitBreaker(inner Sender, maxFailures int, openDuration time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		inner:        inner,
		maxFailures:  maxFailures,
		openDuration: openDuration,
		state:        stateClosed,
	}
}

func (cb *CircuitBreaker) Send(ctx context.Context, msg Message) error {
	cb.mu.Lock()
	switch cb.state {
	case stateOpen:
		if time.Since(cb.openedAt) >= cb.openDuration {
			cb.state = stateHalfOpen
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}
	cb.mu.Unlock()

	err := cb.inner.Send(ctx, msg)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		if cb.failures >= cb.maxFailures {
			cb.state = stateOpen
			cb.openedAt = time.Now()
		}
		return err
	}

	cb.failures = 0
	cb.state = stateClosed
	return nil
}
