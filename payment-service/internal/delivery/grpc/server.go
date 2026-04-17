package grpc

import (
	"context"

	pb "github.com/AQADIL/assignment2-generated/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"payment-service/internal/domain"
)

type Server struct {
	pb.UnimplementedPaymentServiceServer
	uc domain.PaymentUseCase
}

func NewServer(uc domain.PaymentUseCase) *Server {
	return &Server{uc: uc}
}

func (s *Server) ProcessPayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be > 0")
	}

	resp, err := s.uc.CreatePayment(ctx, domain.CreatePaymentRequest{
		OrderID: req.GetOrderId(),
		Amount:  req.GetAmount(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.PaymentResponse{
		PaymentId:     resp.PaymentID,
		TransactionId: resp.TransactionID,
		Status:        resp.Status,
	}, nil
}

func (s *Server) GetPaymentByOrderId(ctx context.Context, req *pb.GetPaymentRequest) (*pb.Payment, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	p, err := s.uc.GetPayment(ctx, req.GetOrderId())
	if err != nil {
		if err == domain.ErrPaymentNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Payment{
		Id:            p.ID,
		OrderId:       p.OrderID,
		TransactionId: p.TransactionID,
		Amount:        p.Amount,
		Status:        p.Status,
	}, nil
}

func (s *Server) ListPayments(ctx context.Context, req *pb.ListPaymentsRequest) (*pb.ListPaymentsResponse, error) {
	payments, err := s.uc.ListPayments(ctx, req.GetMinAmount(), req.GetMaxAmount())
	if err != nil {
		if err == domain.ErrInvalidRange {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &pb.ListPaymentsResponse{}
	for _, p := range payments {
		resp.Payments = append(resp.Payments, &pb.Payment{
			Id:            p.ID,
			OrderId:       p.OrderID,
			TransactionId: p.TransactionID,
			Amount:        p.Amount,
			Status:        p.Status,
		})
	}
	return resp, nil
}
