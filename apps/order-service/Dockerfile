FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.work go.work.sum* ./
COPY apps/order-service/go.mod apps/order-service/go.sum ./apps/order-service/
COPY apps/auth-service/go.mod apps/auth-service/go.sum ./apps/auth-service/

RUN go mod download -C apps/order-service

COPY apps/order-service/ ./apps/order-service/

RUN go build -o order-service ./apps/order-service/main.go

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/order-service .
CMD ["./order-service"]