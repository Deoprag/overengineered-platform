# Estágio de Build
FROM alpine:3.18 AS builder
RUN apk add --no-cache go
WORKDIR /app

# Copia o workspace e os módulos
COPY go.work go.work.sum ./
COPY apps/order-service/go.mod apps/order-service/go.sum ./apps/order-service/
RUN go mod download -C apps/order-service

# Copia o código e compila
COPY apps/order-service/ ./apps/order-service/
RUN go build -o order-api ./apps/order-service/main.go

# Estágio Final (Leve)
FROM alpine:3.18
WORKDIR /root/
COPY --from=builder /app/order-api .
EXPOSE 8081
CMD ["./order-api"]
