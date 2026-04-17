.PHONY: build run-order run-payment clean docker-up docker-down

build:
	cd order-service && go build -o bin/order-service ./cmd/main.go
	cd payment-service && go build -o bin/payment-service ./cmd/main.go

run-order:
	cd order-service && PAYMENT_GRPC_ADDR=localhost:50051 go run ./cmd/main.go

run-payment:
	cd payment-service && go run ./cmd/main.go

clean:
	rm -rf order-service/bin payment-service/bin
	rm -f order-service/order.db payment-service/payment.db
