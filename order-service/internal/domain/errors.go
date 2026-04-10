package domain

import (
	"errors"
)

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrInvalidAmount      = errors.New("amount must be > 0")
	ErrCannotCancelOrder  = errors.New("cannot cancel order in current status")
	ErrPaymentUnavailable = errors.New("payment service unavailable")
)
