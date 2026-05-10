package email

import "context"

type Message struct {
	To      string
	OrderID string
	Amount  int64
}

type Sender interface {
	Send(ctx context.Context, msg Message) error
}
