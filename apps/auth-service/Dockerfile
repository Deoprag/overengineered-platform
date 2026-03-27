# Estágio de Build
FROM alpine:3.18 AS builder
RUN apk add --no-cache go
WORKDIR /app

# Copia o workspace e os módulos
COPY go.work go.work.sum ./
COPY apps/auth-service/go.mod apps/auth-service/go.sum ./apps/auth-service/
RUN go mod download -C apps/auth-service

# Copia o código e compila
COPY apps/auth-service/ ./apps/auth-service/
RUN go build -o auth-api ./apps/auth-service/main.go

# Estágio Final (Leve)
FROM alpine:3.18
WORKDIR /root/
COPY --from=builder /app/auth-api .
EXPOSE 8082
CMD ["./auth-api"]
