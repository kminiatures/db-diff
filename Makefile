.PHONY: build test clean docker-up docker-down docker-reset help

# Detect docker compose command
DOCKER_COMPOSE := $(shell command -v docker-compose 2> /dev/null)
ifndef DOCKER_COMPOSE
	DOCKER_COMPOSE := docker compose
endif

# Build the binary
build:
	go build -o dbdiff ./cmd/dbdiff

# Run tests
test:
	go test ./...

# Run linter
lint:
	go vet ./...

# Clean build artifacts
clean:
	rm -f dbdiff
	rm -rf snapshots/

# Start Docker containers
docker-up:
	$(DOCKER_COMPOSE) up -d

# Stop Docker containers
docker-down:
	$(DOCKER_COMPOSE) down

# Reset Docker containers (remove volumes)
docker-reset:
	$(DOCKER_COMPOSE) down -v
	$(DOCKER_COMPOSE) up -d

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run MySQL test scenario
test-mysql: build
	@echo "Resetting Docker containers..."
	@$(DOCKER_COMPOSE) down -v
	@$(DOCKER_COMPOSE) up -d
	@echo "Waiting for MySQL to be ready..."
	@sleep 10
	@echo "\n=== Creating initial snapshot ==="
	DB_TYPE=mysql DB_HOST=localhost DB_PORT=3306 DB_NAME=testdb DB_USER=testuser DB_PASSWORD=testpass \
		./dbdiff snapshot mysql-before
	@echo "\n=== Running migration ==="
	docker exec -i dbdiff-mysql mysql -utestuser -ptestpass testdb < test/mysql/migration.sql
	@echo "\n=== Creating after snapshot ==="
	DB_TYPE=mysql DB_HOST=localhost DB_PORT=3306 DB_NAME=testdb DB_USER=testuser DB_PASSWORD=testpass \
		./dbdiff snapshot mysql-after
	@echo "\n=== Showing differences ==="
	./dbdiff diff snapshots/mysql-before.db snapshots/mysql-after.db
	@echo "\n=== Generating migration SQL ==="
	./dbdiff migrate snapshots/mysql-before.db snapshots/mysql-after.db

# Run PostgreSQL test scenario
test-postgres: build
	@echo "Resetting Docker containers..."
	@$(DOCKER_COMPOSE) down -v
	@$(DOCKER_COMPOSE) up -d
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 10
	@echo "\n=== Creating initial snapshot ==="
	DB_TYPE=postgres DB_HOST=localhost DB_PORT=5432 DB_NAME=testdb DB_USER=testuser DB_PASSWORD=testpass \
		./dbdiff snapshot postgres-before
	@echo "\n=== Running migration ==="
	docker exec -i dbdiff-postgres psql -U testuser -d testdb < test/postgres/migration.sql
	@echo "\n=== Creating after snapshot ==="
	DB_TYPE=postgres DB_HOST=localhost DB_PORT=5432 DB_NAME=testdb DB_USER=testuser DB_PASSWORD=testpass \
		./dbdiff snapshot postgres-after
	@echo "\n=== Showing differences ==="
	./dbdiff diff snapshots/postgres-before.db snapshots/postgres-after.db
	@echo "\n=== Generating migration SQL ==="
	./dbdiff migrate snapshots/postgres-before.db snapshots/postgres-after.db

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the dbdiff binary"
	@echo "  test          - Run Go tests"
	@echo "  lint          - Run Go linter"
	@echo "  clean         - Clean build artifacts and snapshots"
	@echo "  deps          - Install/update dependencies"
	@echo "  docker-up     - Start Docker containers"
	@echo "  docker-down   - Stop Docker containers"
	@echo "  docker-reset  - Reset Docker containers (remove data)"
	@echo "  test-mysql    - Run complete MySQL test scenario"
	@echo "  test-postgres - Run complete PostgreSQL test scenario"
	@echo "  help          - Show this help message"
