# ==========================================================================
# Cozy Canvas - Development Makefile
# ==========================================================================

ENV_FILE := infrastructure/docker/.env
ifneq ("$(wildcard $(ENV_FILE))","")
    include $(ENV_FILE)
    export
endif

# Default fallback values for database user and name if not set in environment or .env
DB_USER ?= postgres
DB_NAME ?= cozy_canvas


.PHONY: dev stop backend website migrate status clean help test

# Default help menu
help:
	@echo "Cozy Canvas Monorepo Command Panel:"
	@echo "  make dev          - Start Docker infrastructure (PostgreSQL, MinIO)"
	@echo "  make stop         - Stop Docker infrastructure containers"
	@echo "  make backend      - Run the Go REST API locally"
	@echo "  make website      - Run the Vite frontend development server"
	@echo "  make migrate      - Apply SQL migrations to PostgreSQL"
	@echo "  make test         - Run integration API tests"
	@echo "  make status       - Display running Docker services"

# Start PostgreSQL and MinIO services using Docker Compose
dev:
	docker compose -f infrastructure/docker/docker-compose.yml -f infrastructure/docker/docker-compose.override.yml up -d
	@echo "🌸 Cozy Canvas local infrastructure started successfully!"

# Stop all running containers while preserving volumes
stop:
	docker compose -f infrastructure/docker/docker-compose.yml down
	@echo "🌸 Cozy Canvas infrastructure services stopped."

# Run the Go REST API server locally
backend:
	@echo "🌸 Starting Go API server..."
	cd backend && go run cmd/api/main.go

# Start the Vite frontend development server
website:
	@echo "🌸 Starting Vite frontend server..."
	cd website && npm install && npm run dev

# Apply database migrations to PostgreSQL running inside Docker
# FIXED: Database user (-U) and database name (-d) are now loaded from .env
migrate:
	@echo "🌸 Applying users migration to [$(DB_NAME)] as user [$(DB_USER)]..."
	docker compose -f infrastructure/docker/docker-compose.yml exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -f /migrations/001_create_users.sql
	@echo "🌸 Applying canvas nodes and connections migration..."
	docker compose -f infrastructure/docker/docker-compose.yml exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -f /migrations/002_create_nodes.sql
	@echo "🌸 Applying access grants migration..."
	docker compose -f infrastructure/docker/docker-compose.yml exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -f /migrations/003_access_grants.sql
	@echo "🌸 All database migrations applied successfully!"

# Display the status of running Docker containers
status:
	docker compose -f infrastructure/docker/docker-compose.yml ps

# Run API integration tests
test:
	@echo "🌸 Running integration API tests..."
	bash tests/api/test_all.sh