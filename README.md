# AP2 Assignment 1 — Order Service + Payment Service (Clean Architecture)

## Project Overview
This repository contains two isolated Go microservices:

- **Order Service** (`order-service`, port **8080**) — manages orders, persists them in SQLite, calls Payment Service synchronously over REST, implements idempotent order creation.
- **Payment Service** (`payment-service`, port **8081**) — authorizes/declines payments based on a strict amount rule and persists results in its own SQLite database.

Each service is a **separate Go module** with **no shared packages**.

## Architecture (Strict Clean Architecture)
Each service is split into layers:

- **Delivery** (`internal/delivery/http`) — thin Gin HTTP layer (routing + handlers).
- **Use Case** (`internal/usecase`) — business logic and orchestration.
- **Repository** (`internal/repository`) — SQLite persistence + auto-migrations.
- **Domain** (`internal/domain`) — entities, DTOs, ports (interfaces), domain errors.

Dependencies flow inward:

`delivery -> usecase -> domain <- repository`

The repository and external clients implement ports defined in the domain.

## Bounded Contexts / Isolation
- The **Order Service** defines its own `Order` entity, DTOs, and ports.
- The **Payment Service** defines its own `Payment` entity, DTOs, and ports.
- There is **no shared folder** and **no import from one service into the other**.

## Money Representation
All money values are `int64` representing **cents**. No floats are used.

## Order Service Details
### Endpoints
- `POST /orders`
  - Saves order as `Pending`.
  - Calls Payment Service (`POST /payments`) synchronously.
  - Updates order status to:
    - `Paid` if Payment returns `Authorized`
    - `Failed` otherwise
- `GET /orders/:id`
- `PATCH /orders/:id/cancel` — only `Pending` orders can be cancelled.

### Timeout / Failure Handling
The Payment client in Order Service uses an **HTTP client timeout of 2 seconds**.

If the Payment Service is down or times out:
- Order Service **updates the order to `Failed`** in SQLite.
- Order Service returns **HTTP 503 Service Unavailable**.

This ensures Order Service never hangs indefinitely.

### Idempotency
Order creation supports idempotency via the `Idempotency-Key` header:
- A Gin middleware checks `idempotency_keys` table.
- If key exists, the stored `(status_code, response_body)` is replayed.
- Otherwise, after processing request, the response is stored.

## Payment Service Details
### Endpoints
- `POST /payments` — creates a payment decision for an order.
- `GET /payments/:order_id` — fetches payment by order id.

### Business Rule
If `amount > 100000` then status is `Declined`, else `Authorized`.

## Running Locally
### Option A: Run directly
In two terminals:

- Order Service:
  - `make run-order`
- Payment Service:
  - `make run-payment`

### Option B: Docker Compose
- `docker compose up`

Order Service is configured with:
- `PAYMENT_URL=http://payment-service:8081`

## Build
- `make build`

## Clean
- `make clean`

## Packaging into ZIP
To package the final source code into a zip archive named exactly:

`AP2_Assignment1_Alish_Akadil_SE-2426.zip`

Use any method you prefer, for example from the repository root:

- Windows PowerShell:
  - `Compress-Archive -Path .\ap2_assignment1\* -DestinationPath .\AP2_Assignment1_Alish_Akadil_SE-2426.zip`

Ensure the zip contains the project root with `order-service/`, `payment-service/`, `Makefile`, `docker-compose.yml`, and `README.md`.
