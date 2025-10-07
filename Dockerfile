# Multi-stage build для оптимизации размера образа
FROM golang:1.21-alpine AS builder

# Устанавливаем рабочую директорию
WORKDIR /app

# Устанавливаем необходимые пакеты
RUN apk add --no-cache git

# Копируем go.mod и go.sum
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Генерируем Swagger документацию
RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init -g cmd/server/main.go -o docs/

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Финальный образ
FROM alpine:latest

# Устанавливаем ca-certificates для HTTPS запросов
RUN apk --no-cache add ca-certificates postgresql-client

WORKDIR /root/

# Копируем бинарный файл из builder stage
COPY --from=builder /app/main .

# Копируем Swagger документацию
COPY --from=builder /app/docs ./docs/

# Открываем порт
EXPOSE 8080

# Переменные окружения по умолчанию
ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=8080
ENV SERVER_READ_TIMEOUT=15s
ENV SERVER_WRITE_TIMEOUT=15s
ENV SERVER_IDLE_TIMEOUT=60s
ENV DB_HOST=postgres
ENV DB_PORT=5432
ENV DB_USER=postgres
ENV DB_PASSWORD=postgres
ENV DB_NAME=currency_quotes
ENV DB_SSLMODE=disable
ENV LOG_LEVEL=info
ENV LOG_FORMAT=json
ENV EXTERNAL_API_URL=https://api.fxratesapi.com
ENV EXTERNAL_API_TIMEOUT=10s
ENV EXTERNAL_API_KEY=
ENV WORKER_INTERVAL=30s
ENV SHUTDOWN_TIMEOUT=30s
ENV SUPPORTED_CURRENCIES=USD,EUR,MXN

# Запускаем приложение
CMD ["./main"]
