# ─────────────────────────────────────────────
# Stage 1: Build
# ─────────────────────────────────────────────
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o tiket-bps-api \
    ./main.go

# ─────────────────────────────────────────────
# Stage 2: Runtime
# ─────────────────────────────────────────────
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Makassar /etc/localtime && \
    echo "Asia/Makassar" > /etc/timezone

WORKDIR /app

COPY --from=builder /app/tiket-bps-api .
COPY --from=builder /app/public ./public/

EXPOSE 8080

CMD ["./tiket-bps-api"]
