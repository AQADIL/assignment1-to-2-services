package grpc

import (
	"context"

	pb "github.com/AQADIL/assignment2-generated/order"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"order-service/internal/domain"
)

type Server struct {
	pb.UnimplementedOrderServiceServer
	uc   domain.OrderUseCase
	repo domain.OrderRepository
}

func NewServer(uc domain.OrderUseCase, repo domain.OrderRepository) *Server {
	return &Server{uc: uc, repo: repo}
}

func (s *Server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.OrderResponse, error) {
	if req.GetCustomerId() == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id is required")
	}
	if req.GetItemName() == "" {
		return nil, status.Error(codes.InvalidArgument, "item_name is required")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be > 0")
	}

	order, err := s.uc.CreateOrder(ctx, domain.CreateOrderRequest{
		CustomerID: req.GetCustomerId(),
		ItemName:   req.GetItemName(),
		Amount:     req.GetAmount(),
	})
	if err != nil {
		if err == domain.ErrInvalidAmount {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if err == domain.ErrPaymentUnavailable {
			return nil, status.Error(codes.Unavailable, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return orderToProto(order), nil
}

func (s *Server) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	order, err := s.uc.GetOrder(ctx, req.GetOrderId())
	if err != nil {
		if err == domain.ErrOrderNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return orderToProto(order), nil
}

func (s *Server) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.OrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	order, err := s.uc.CancelOrder(ctx, req.GetOrderId())
	if err != nil {
		if err == domain.ErrOrderNotFound {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if err == domain.ErrCannotCancelOrder {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return orderToProto(order), nil
}

func (s *Server) ListOrdersByCustomer(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	if req.GetCustomerId() == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id is required")
	}

	orders, err := s.uc.ListOrdersByCustomer(ctx, req.GetCustomerId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &pb.ListOrdersResponse{}
	for _, o := range orders {
		resp.Orders = append(resp.Orders, orderToProto(o))
	}
	return resp, nil
}

func (s *Server) SubscribeToOrderUpdates(req *pb.OrderFilter, stream pb.OrderService_SubscribeToOrderUpdatesServer) error {
	ch, subID := s.repo.Subscribe(req.GetCustomerId())
	defer s.repo.Unsubscribe(subID)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case order, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(orderToProto(order)); err != nil {
				return err
			}
		}
	}
}

func orderToProto(o domain.Order) *pb.OrderResponse {
	return &pb.OrderResponse{
		OrderId:    o.ID,
		CustomerId: o.CustomerID,
		ItemName:   o.ItemName,
		Amount:     o.Amount,
		Status:     o.Status,
		CreatedAt:  o.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
