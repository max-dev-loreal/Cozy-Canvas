# ==========================================================================
# Cozy Canvas - Development Master Makefile
# ==========================================================================

.PHONY: dev stop backend website migrate status clean help

# Default help screen
help:
	@echo "Cozy Canvas Monorepo Command Panel:"
	@echo "  make dev          - Start Docker infrastructure (Postgres, MinIO)"
	@echo "  make stop         - Shutdown Docker infrastructure containers"
	@echo "  make backend      - Run Go REST API backend locally"
	@echo "  make website      - Run Vite Frontend developer server"
	@echo "  make migrate      - Apply SQL migrations into Docker Postgres container"
	@echo "  make status       - Inspect running Docker services status"

# Run PostgreSQL & MinIO Docker Compose services
dev:
	docker compose -f infrastructure/docker/docker-compose.yml -f infrastructure/docker/docker-compose.override.yml up -d
	@echo "🌸 Cozy Canvas local infrastructure started successfully!"

# Stop all running containers and preserve volumes
stop:
	docker compose -f infrastructure/docker/docker-compose.yml down
	@echo "🌸 Cozy Canvas infrastructure services stopped."

# Compile and run the Go REST API Server
backend:
	@echo "🌸 Starting Go API Server..."
	cd backend && go run cmd/api/main.go

# Start the Vite developer server for the canvas frontend
website:
	@echo "🌸 Starting Vite Frontend server..."
	cd website && npm install && npm run dev

# Apply DB migrations directly to PostgreSQL inside the container
migrate:
	@echo "🌸 Applying users database migration..."
	docker compose -f infrastructure/docker/docker-compose.yml exec -T postgres psql -U postgres -d cozy_canvas -f /migrations/001_create_users.sql
	@echo "🌸 Applying canvas nodes and connections migration..."
	docker compose -f infrastructure/docker/docker-compose.yml exec -T postgres psql -U postgres -d cozy_canvas -f /migrations/002_create_nodes.sql
	@echo "🌸 All SQL database migrations successfully applied!"

# View Docker containers status
status:
	docker compose -f infrastructure/docker/docker-compose.yml ps
