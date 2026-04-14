.PHONY: up down build logs clean test seed

# Start everything
up:
	docker compose up --build -d
	@echo "✅  App:  http://localhost:3000"
	@echo "✅  API:  http://localhost:8080"

# Start in foreground
dev:
	docker compose up --build

# Stop everything
down:
	docker compose down

# Remove volumes (wipes DB)
clean:
	docker compose down -v

# View logs
logs:
	docker compose logs -f

# Backend logs only
logs-api:
	docker compose logs -f backend

# Run integration tests (requires running postgres)
test:
	cd backend && \
	TEST_DB_HOST=localhost \
	TEST_DB_USER=$$(grep DB_USER .env | cut -d= -f2) \
	TEST_DB_PASSWORD=$$(grep DB_PASSWORD .env | cut -d= -f2) \
	TEST_DB_NAME=$$(grep DB_NAME .env | cut -d= -f2) \
	go test ./... -v

# Re-run seed data
seed:
	docker compose exec postgres psql -U $$(grep DB_USER .env | cut -d= -f2) \
		-d $$(grep DB_NAME .env | cut -d= -f2) \
		-f /docker-entrypoint-initdb.d/seed.sql

# Build images only (no start)
build:
	docker compose build
