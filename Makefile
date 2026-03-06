.PHONY: help up down logs ps dev test lint clean migrate seed docker-build docker-push

# Default target
help:
	@echo "E-Commerce Platform - Development Commands"
	@echo ""
	@echo "Infrastructure:"
	@echo "  make up              - Start all containers (PostgreSQL, Kafka, Redis, etc.)"
	@echo "  make down            - Stop all containers"
	@echo "  make logs            - Show container logs"
	@echo "  make ps              - Show running containers"
	@echo ""
	@echo "Development:"
	@echo "  make dev             - Start services with hot-reload (air)"
	@echo "  make test            - Run all tests with coverage"
	@echo "  make lint            - Run linters (golangci-lint, gosec)"
	@echo "  make clean           - Clean build artifacts"
	@echo ""
	@echo "Database:"
	@echo "  make migrate         - Run database migrations"
	@echo "  make seed            - Seed development data"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build    - Build all service Docker images"
	@echo "  make docker-push     - Push images to ECR"
	@echo ""

# Infrastructure targets
up:
	docker-compose up -d
	@echo "Waiting for services to be healthy..."
	@sleep 10
	@echo "Services are running!"

down:
	docker-compose down

logs:
	docker-compose logs -f

ps:
	docker-compose ps

# Development targets
dev:
	@command -v air >/dev/null 2>&1 || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	cd services/user-service && air &
	cd services/api-gateway && air &
	wait

test:
	@echo "Running tests for shared..."
	cd shared && go test -v -race -coverprofile=coverage.out ./...
	@echo ""
	@echo "Running tests for user-service..."
	cd services/user-service && go test -v -race -coverprofile=coverage.out ./...
	@echo ""
	@echo "Running tests for api-gateway..."
	cd services/api-gateway && go test -v -race -coverprofile=coverage.out ./...
	@echo ""
	@echo "Test coverage reports generated"

lint:
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@command -v gosec >/dev/null 2>&1 || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	
	@echo "Linting shared..."
	cd shared && golangci-lint run ./... && gosec ./...
	
	@echo "Linting user-service..."
	cd services/user-service && golangci-lint run ./... && gosec ./...
	
	@echo "Linting api-gateway..."
	cd services/api-gateway && golangci-lint run ./... && gosec ./...
	
	@echo "All linters passed!"

clean:
	@echo "Cleaning build artifacts..."
	find . -name "*.out" -delete
	find . -path "*/bin/*" -type f -delete
	rm -f air*.log
	@echo "Cleanup complete"

# Database targets
migrate:
	@echo "Running database migrations..."
	docker-compose exec postgres psql -U ecommerce -d ecommerce -a -f /docker-entrypoint-initdb.d/001_init_users.sql
	docker-compose exec postgres psql -U ecommerce -d ecommerce -a -f /docker-entrypoint-initdb.d/002_init_products_inventory.sql
	docker-compose exec postgres psql -U ecommerce -d ecommerce -a -f /docker-entrypoint-initdb.d/003_init_orders_payments.sql
	@echo "Migrations complete"

seed:
	@echo "Seeding database..."
	go run ./scripts/seed/main.go
	@echo "Seeding complete"

# Docker targets
docker-build:
	@echo "Building Docker images..."
	docker build -t ecommerce/user-service:latest -f services/user-service/Dockerfile .
	docker build -t ecommerce/api-gateway:latest -f services/api-gateway/Dockerfile .
	@echo "Docker builds complete"

docker-push:
	@echo "Pushing images to ECR..."
	aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin $(AWS_ACCOUNT_ID).dkr.ecr.us-east-1.amazonaws.com
	docker tag ecommerce/user-service:latest $(AWS_ACCOUNT_ID).dkr.ecr.us-east-1.amazonaws.com/ecommerce/user-service:latest
	docker tag ecommerce/api-gateway:latest $(AWS_ACCOUNT_ID).dkr.ecr.us-east-1.amazonaws.com/ecommerce/api-gateway:latest
	docker push $(AWS_ACCOUNT_ID).dkr.ecr.us-east-1.amazonaws.com/ecommerce/user-service:latest
	docker push $(AWS_ACCOUNT_ID).dkr.ecr.us-east-1.amazonaws.com/ecommerce/api-gateway:latest
	@echo "Push complete"
