# syntax=docker/dockerfile:1

# ===== Build stage =====
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Установка зависимостей для CGO (требуется для SQLite)
RUN apk add --no-cache git ca-certificates gcc musl-dev sqlite-dev

# Модульные файлы
COPY go.mod go.sum ./
RUN go mod download

# Исходники бэка (контекст ./backend)
COPY . .

# Собираем сервер с CGO_ENABLED=1 (требуется для go-sqlite3)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /out/quiz-server ./cmd/server

# ===== Runtime stage =====
FROM alpine:3.20
WORKDIR /srv

# Установка runtime зависимостей для SQLite
RUN apk add --no-cache ca-certificates tzdata sqlite

# Кладём бинарь
COPY --from=builder /out/quiz-server /srv/quiz-server

# Создаём директорию для БД
RUN mkdir -p /srv/data && chmod 755 /srv/data

# Слушаем 8081 (без TLS — TLS на nginx хосте)
EXPOSE 8081

# Healthcheck
HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD wget -q -O- http://127.0.0.1:8081/health || exit 1

ENTRYPOINT ["/srv/quiz-server"]
