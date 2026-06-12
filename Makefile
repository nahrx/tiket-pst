APP_NAME   = tiket-bps-api
BUILD_DIR  = ./bin
MAIN       = ./main.go

.PHONY: all build run dev clean test tidy docker-build docker-run compose-up compose-down

all: build

## Build binary
build:
	@echo "→ Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN)
	@echo "✓ Binary: $(BUILD_DIR)/$(APP_NAME)"

## Run langsung dengan go run
run:
	go run $(MAIN)

## Run dengan hot reload (butuh: go install github.com/air-verse/air@latest)
dev:
	air

## Jalankan tests
test:
	go test ./... -v

## Tidy modules
tidy:
	go mod tidy

## Bersihkan build artifacts
clean:
	rm -rf $(BUILD_DIR) ./tmp

## Build Docker image
docker-build:
	docker build -t $(APP_NAME):latest .

## Run Docker container (sambungkan ke PostgreSQL eksternal)
docker-run:
	docker run -p 8080:8080 \
		-e JWT_SECRET=secret_production_min32chars \
		-e APP_ENV=production \
		-e DB_HOST=host.docker.internal \
		-e DB_NAME=tiket_bps \
		-e DB_USER=postgres \
		-e DB_PASSWORD=postgres \
		$(APP_NAME):latest

## Start dengan Docker Compose (PostgreSQL + API)
compose-up:
	docker compose up -d --build
	@echo "✓ API: http://localhost:8080"
	@echo "✓ DB:  localhost:5432"

## Stop Docker Compose
compose-down:
	docker compose down

## Tampilkan semua endpoint
routes:
	@echo ""
	@echo "Public Endpoints:"
	@echo "  GET  /api/health"
	@echo "  GET  /api/cek-hp?no_hp=08xx"
	@echo "  POST /api/register"
	@echo "  POST /api/tiket"
	@echo "  POST /api/login"
	@echo ""
	@echo "Protected Endpoints (Bearer JWT):"
	@echo "  GET   /api/admin/profile"
	@echo "  GET   /api/admin/stats"
	@echo "  GET   /api/admin/tiket[?status=baru&page=1&limit=20&q=keyword]"
	@echo "  GET   /api/admin/tiket/{nomor_tiket}"
	@echo "  PATCH /api/admin/tiket/{nomor_tiket}/status"
	@echo ""
