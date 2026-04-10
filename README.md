# AP2 — Assignment 2: Order & Payment Microservices (gRPC + Clean Architecture)

## Project Overview

This repository contains two isolated Go microservices communicating via **gRPC**:

| Service | Transport | Port(s) | Database |
|---------|-----------|---------|----------|
| **Order Service** | HTTP (Gin) + gRPC | HTTP `8080`, gRPC `50052` | SQLite `order.db` |
| **Payment Service** | gRPC only | gRPC `50051` | SQLite `payment.db` |

Each service is a **separate Go module** with **no shared packages**.

## Remote Proto Repositories

| Repository | Purpose |
|------------|---------|
| [assignment2-protos](https://github.com/AQADIL/assignment2-protos) | `.proto` source definitions for both services |
| [assignment2-generated](https://github.com/AQADIL/assignment2-generated) | Generated Go code (`*.pb.go` + `*_grpc.pb.go`) imported by both services |

The generated Go module is imported as:
```go
import pb "github.com/AQADIL/assignment2-generated/order"
import pb "github.com/AQADIL/assignment2-generated/payment"
```

## Architecture (Strict Clean Architecture)

Each service is split into layers:

```
internal/
├── domain/         # Entities, DTOs, ports (interfaces), domain errors
├── usecase/        # Business logic and orchestration
├── repository/     # SQLite persistence + auto-migrations
└── delivery/
    ├── http/       # Gin REST layer (Order Service only)
    └── grpc/       # gRPC server implementation
```

Dependencies flow inward: `delivery → usecase → domain ← repository`

The Domain and Use Case layers are **never modified** for transport changes — only the Delivery layer was swapped from REST to gRPC.

## Bounded Contexts / Isolation

- **Order Service** defines its own `Order` entity, DTOs, and ports.
- **Payment Service** defines its own `Payment` entity, DTOs, and ports.
- There is **no shared folder** and **no import from one service into the other**.
- Inter-service communication uses the shared protobuf contracts from `assignment2-generated`.

## Money Representation

All money values are `int64` representing **cents**. No floats are used.

---

## Order Service

### gRPC RPCs (port 50052)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `CreateOrder` | `CreateOrderRequest` | `OrderResponse` | Create order, call Payment Service via gRPC |
| `GetOrder` | `GetOrderRequest` | `OrderResponse` | Get order by ID |
| `CancelOrder` | `CancelOrderRequest` | `OrderResponse` | Cancel a Pending order |
| `ListOrdersByCustomer` | `ListOrdersRequest` | `ListOrdersResponse` | List all orders for a customer |
| `SubscribeToOrderUpdates` | `OrderFilter` | `stream OrderResponse` | **Server-streaming** real-time order updates |

### HTTP REST Endpoints (port 8080, backward compatible)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/orders` | Create order (with `Idempotency-Key` header) |
| `GET` | `/orders/:id` | Get order by ID |
| `GET` | `/orders?customer_id=xxx` | List orders by customer |
| `PATCH` | `/orders/:id/cancel` | Cancel order |

### Payment Client (gRPC)

The Order Service calls Payment Service via **gRPC** with a **2-second `context.WithTimeout`**:
```go
ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
resp, err := client.ProcessPayment(ctx, &pb.PaymentRequest{...})
```

If Payment Service is unreachable or times out:
- Order status is updated to `Failed`.
- gRPC returns `codes.Unavailable`; HTTP returns `503`.

### Real-Time Streaming (`SubscribeToOrderUpdates`)

This uses **Go channels tied to the SQLite repository** — no fake `time.Sleep` loops:

1. Repository maintains a `map[string]*subscriber` with buffered channels.
2. When `UpdateStatus()` is called, `notify()` pushes the updated `Order` to all matching subscribers.
3. The gRPC stream handler reads from the channel and sends to the client.
4. On client disconnect (`stream.Context().Done()`), the subscriber is cleaned up.

```
Client ──SubscribeToOrderUpdates──▶ gRPC Server
                                        │
                                   Subscribe(customerID)
                                        │
                                   ◀── ch <-chan Order
                                        │
    UpdateStatus() ──▶ notify() ──▶ ch ──▶ stream.Send()
```

### Idempotency (HTTP only)

Order creation via HTTP supports idempotency via the `Idempotency-Key` header:
- Middleware checks `idempotency_keys` table using `INSERT OR IGNORE`.
- If key exists, the stored `(status_code, response_body)` is replayed.

---

## Payment Service

### gRPC RPCs (port 50051)

| RPC | Request | Response | Description |
|-----|---------|----------|-------------|
| `ProcessPayment` | `PaymentRequest` | `PaymentResponse` | Authorize or decline a payment |
| `GetPaymentByOrderId` | `GetPaymentRequest` | `Payment` | Get payment by order ID |

### Business Rule

If `amount > 100000` → status `Declined`, else `Authorized`.

### Logging Interceptor (Bonus)

A **gRPC UnaryServerInterceptor** logs every RPC call:
```go
func LoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
    return func(ctx, req, info, handler) (resp, err) {
        start := time.Now()
        resp, err = handler(ctx, req)
        logger.Info("grpc_request",
            "method", info.FullMethod,
            "duration_ms", time.Since(start).Milliseconds(),
            "error", err,
        )
        return resp, err
    }
}
```

### Error Handling

Both services use `google.golang.org/grpc/status` and `codes`:

| Domain Error | gRPC Code |
|-------------|-----------|
| `ErrOrderNotFound` / `ErrPaymentNotFound` | `codes.NotFound` |
| `ErrInvalidAmount` | `codes.InvalidArgument` |
| `ErrCannotCancelOrder` | `codes.FailedPrecondition` |
| `ErrPaymentUnavailable` | `codes.Unavailable` |
| Internal errors | `codes.Internal` |

---

## Environment Variables

### Order Service

| Variable | Default | Description |
|----------|---------|-------------|
| `ORDER_DB` | `./order.db` | Path to SQLite database |
| `PAYMENT_GRPC_ADDR` | `localhost:50051` | Payment Service gRPC address |
| `HTTP_PORT` | `8080` | HTTP server port |
| `GRPC_PORT` | `50052` | gRPC server port |

### Payment Service

| Variable | Default | Description |
|----------|---------|-------------|
| `PAYMENT_DB` | `./payment.db` | Path to SQLite database |
| `GRPC_PORT` | `50051` | gRPC server port |

---

## Running

### Option A: Docker Compose (recommended)

```bash
docker compose up -d
docker compose logs -f
```

### Option B: Locally (two terminals)

```bash
# Terminal 1 — Payment Service
make run-payment

# Terminal 2 — Order Service
make run-order
```

### Testing with grpcurl

```bash
# Create a payment
grpcurl -plaintext -d '{"order_id":"test-1","amount":5000}' localhost:50051 payment.PaymentService/ProcessPayment

# Create an order
grpcurl -plaintext -d '{"customer_id":"alice","item_name":"Laptop","amount":50000}' localhost:50052 order.OrderService/CreateOrder

# Get an order
grpcurl -plaintext -d '{"order_id":"<UUID>"}' localhost:50052 order.OrderService/GetOrder

# List orders by customer
grpcurl -plaintext -d '{"customer_id":"alice"}' localhost:50052 order.OrderService/ListOrdersByCustomer

# Subscribe to real-time updates (streams)
grpcurl -plaintext -d '{"customer_id":"alice"}' localhost:50052 order.OrderService/SubscribeToOrderUpdates
```

## Build

```bash
make build
```

## Clean

```bash
make clean
```
