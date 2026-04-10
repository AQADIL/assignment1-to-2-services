package paymentclient

import (
	"context"
	"time"

	pb "github.com/AQADIL/assignment2-generated/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"order-service/internal/domain"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.PaymentServiceClient
}

func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:   conn,
		client: pb.NewPaymentServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) CreatePayment(ctx context.Context, req domain.PaymentCreateRequest) (domain.PaymentCreateResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := c.client.ProcessPayment(ctx, &pb.PaymentRequest{
		OrderId: req.OrderID,
		Amount:  req.Amount,
	})
	if err != nil {
		return domain.PaymentCreateResponse{}, err
	}

	return domain.PaymentCreateResponse{
		PaymentID:     resp.GetPaymentId(),
		TransactionID: resp.GetTransactionId(),
		Status:        resp.GetStatus(),
	}, nil
}
