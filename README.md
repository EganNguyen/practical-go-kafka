# Event-Driven Microservices E-Commerce Platform

An enterprise-grade e-commerce platform built with Go, Kafka, and Kubernetes. The system implements CQRS pattern for order management and event sourcing for reliable, distributed transaction processing.

**Status:** Phase 1 — Core Foundation ✅

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.22+
- PostgreSQL 16 (via Docker)
- Kafka 3.x (via Docker)
- Redis 7 (via Docker)

### Local Development Setup

1. **Clone and setup:**
```bash
git clone <repo>
cd practical-go-kafka
```

2. **Start infrastructure:**
```bash
make up
```

This starts:
- PostgreSQL (port 5432)
- MongoDB (port 27017)
- Redis (port 6379)
- Kafka (port 9092)
- Elasticsearch (port 9200)
- Zookeeper (port 2181)

3. **Run migrations:**
```bash
make migrate
```

4. **Start services (with hot-reload):**
```bash
make dev
```

Services start on:
- API Gateway: http://localhost:8080
- User Service: http://localhost:8081
- Product Service: http://localhost:8082
- Cart Service: http://localhost:8084
- Order Service: http://localhost:8085
- Payment Service: http://localhost:8086
- Search Service: http://localhost:8088

### Testing

```bash
# Run all tests with coverage
make test

# Run linters
make lint

# Run specific service tests
cd services/user-service
go test -v -race ./...
```

## Project Structure

```
├── services/           # 10 microservices (Go + Gin)
│   ├── api-gateway/    # Single ingress point
│   ├── user-service/   # Auth & user management
│   ├── product-service/# Product catalog (MongoDB)
│   ├── inventory-service/ # Stock management
│   ├── cart-service/   # Shopping cart (Redis)
│   ├── order-service/  # Order processing (CQRS)
│   ├── payment-service/# Stripe/PayPal integration
│   ├── notification-service/ # Email/SMS/push
│   ├── search-service/ # Elasticsearch integration
│   └── analytics-service/    # Event aggregation
├── shared/             # Go modules (events, models, JWT)
│   ├── events/         # Kafka envelope & producer/consumer
│   ├── model/          # Domain models (User, Order, etc.)
│   └── pkg/            # Common utilities (JWT, config)
├── frontend/           # Next.js 14 storefront
├── infra/              # Infrastructure & deployment
│   ├── terraform/      # AWS infrastructure (EKS, RDS, MSK, ElastiCache)
│   ├── helm/           # Kubernetes Helm charts
│   ├── k8s/            # Raw Kubernetes manifests
│   └── migrations/     # Database migration scripts
├── .github/workflows/  # GitHub Actions CI/CD pipeline
├── docker-compose.yml  # Local development environment
├── Makefile            # Development commands
└── ecommerce-platform-spec.md # Master specification document
```

## Technology Stack

| Component | Technology |
|-----------|------------|
| **Language** | Go 1.22+ |
| **Framework** | Gin (HTTP), Gorilla Mux (gateway) |
| **Message Bus** | Apache Kafka 3.x |
| **Relational DB** | PostgreSQL 16 |
| **Document DB** | MongoDB 7 |
| **Cache/Session** | Redis 7 |
| **Search** | Elasticsearch 8.x |
| **Container** | Docker + Kubernetes (EKS) |
| **IaC** | Terraform |
| **CI/CD** | GitHub Actions |
| **Observability** | Prometheus + Grafana + ELK |
| **Frontend** | Next.js 14 + React 18 + Tailwind |

## Phase 1 Implementation

### ✅ Completed

1. **Monorepo Structure**
   - Organized service directories with standard Go layouts
   - Shared modules for common code (events, models, utilities)

2. **Shared Libraries**
   - Event envelope implementation (Kafka message format)
   - JWT manager with RS256 signing/verification
   - Config loader with environment variable support
   - Domain models (User, Order, Payment, etc.)

3. **User Service**
   - Registration, login, refresh token, logout
   - JWT-based authentication (15m access / 7d refresh)
   - User profile management
   - PostgreSQL repository layer
   - Password hashing (bcrypt cost 12)

4. **API Gateway**
   - Request ID injection for tracing
   - JWT validation middleware
   - Rate limiting (sliding window in Redis)
   - CORS policy enforcement
   - Forward to downstream services
   - Health check endpoint

5. **Infrastructure**
   - **Terraform:** VPC, EKS clusters, RDS (Aurora PostgreSQL), MSK, ElastiCache, ECR
   - **Helm Charts:** Service templates, umbrella chart for deployment
   - **Database Migrations:** User, inventory, order, payment schemas
   - **Docker:** Multi-stage builds for User Service and API Gateway

6. **CI/CD Pipeline**
   - GitHub Actions workflow (lint → test → build → deploy)
   - Pull request validation with 80% coverage requirement
   - Security scanning (gosec, Trivy)
   - Staging and production deployments
   - Blue-green deployment support

7. **Local Development**
   - Docker Compose with PostgreSQL, MongoDB, Redis, Kafka, Elasticsearch
   - Makefile with convenience commands
   - Hot-reload support (air)

## API Documentation

### Authentication

**Register**
```bash
POST /v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure-password",
  "first_name": "John",
  "last_name": "Doe"
}
```

**Login**
```bash
POST /v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secure-password"
}
```

Response:
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

**Refresh Token**
```bash
POST /v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJ..."
}
```

### Protected Endpoints

```bash
# Get user profile
GET /v1/users/me
Authorization: Bearer <access_token>

# Update profile
PATCH /v1/users/me
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "first_name": "Jane",
  "last_name": "Smith"
}

# Delete account
DELETE /v1/users/me
Authorization: Bearer <access_token>
```

## Event Schema

All Kafka events follow this envelope:

```json
{
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "order.created",
  "aggregate_id": "550e8400-e29b-41d4-a716-446655440001",
  "aggregate_type": "order",
  "version": 1,
  "timestamp": "2026-03-06T12:00:00Z",
  "correlation_id": "550e8400-e29b-41d4-a716-446655440002",
  "producer_service": "order-service",
  "payload": {
    "order_id": "550e8400-e29b-41d4-a716-446655440001",
    "user_id": "550e8400-e29b-41d4-a716-446655440003",
    "total_amount": 99.99,
    "currency": "USD"
  }
}
```

## Development Workflow

### Adding a New Service

1. Create service directory: `services/new-service/`
2. Copy structure: `cmd/server/`, `internal/{handler,service,repository,config,middleware}`
3. Add to `go.work` file
4. Create `Dockerfile` and `.air.toml` for development
5. Add Helm deployment to `infra/helm/templates/`
6. Add CI/CD step to `.github/workflows/main.yml`

### Running Tests with Coverage

```bash
# All services
make test

# Single service
cd services/user-service
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Database Migrations

Migrations are SQL files in `infra/migrations/`:

```bash
# Apply migrations
make migrate

# Add new migration: infra/migrations/004_add_new_feature.sql
```

## Configuration

All services use environment variables for configuration:

```bash
# User Service
USER_SERVICE_PORT=8081
USER_SERVICE_DATABASE_URL=postgres://user:pass@localhost:5432/ecommerce
USER_SERVICE_KAFKA_BROKERS=localhost:9092
USER_SERVICE_JWT_PRIVATE_KEY_PATH=/etc/secrets/jwt_private_key
USER_SERVICE_JWT_PUBLIC_KEY_PATH=/etc/secrets/jwt_public_key
```

See service `internal/config/config.go` for all available options.

## Deployment

### Kubernetes (Helm)

```bash
# Dry-run
helm install ecommerce-platform ./infra/helm --dry-run

# Install to production
helm install ecommerce-platform ./infra/helm \
  --namespace ecommerce \
  --create-namespace \
  --values infra/helm/values-prod.yaml

# Upgrade
helm upgrade ecommerce-platform ./infra/helm \
  --namespace ecommerce \
  --wait --atomic
```

### AWS Infrastructure (Terraform)

```bash
cd infra/terraform

# Initialize
terraform init

# Plan
terraform plan -var-file=environments/prod.tfvars

# Apply
terraform apply -var-file=environments/prod.tfvars
```

## Monitoring & Observability

- **Metrics:** Prometheus scrapes `/metrics` endpoint on each service (port+1)
- **Logging:** Structured JSON logs via zerolog, shipped to CloudWatch/ELK
- **Tracing:** X-Request-ID header propagation via middleware
- **Dashboards:** Grafana dashboards in `docs/dashboards/`

## Performance Targets (SLA)

| Endpoint | p50 | p99 | SLA |
|----------|-----|-----|-----|
| Product page (SSR) | 80 ms | 200 ms | 99.9% |
| Search query | 30 ms | 100 ms | 99.9% |
| Cart GET (Redis) | 5 ms | 20 ms | 99.99% |
| Order creation | 100 ms | 400 ms | 99.9% |
| Platform availability | — | — | **99.99%** |

## Security Considerations

- ✅ All external traffic TLS 1.3
- ✅ JWT RS256 with quarterly key rotation
- ✅ Parameterized SQL queries (no injection)
- ✅ Password hashing: bcrypt cost 12
- ✅ Secrets in AWS Secrets Manager
- ✅ OWASP Top 10 compliance via gosec/gitleaks
- ✅ Container scanning with Trivy

## Troubleshooting

### Services won't start

```bash
# Check logs
make logs

# Verify database connection
docker-compose exec postgres psql -U ecommerce -d ecommerce -c "SELECT 1;"

# Check Kafka
docker-compose exec kafka kafka-topics.sh --bootstrap-server=localhost:9092 --list
```

### Port conflicts

```bash
# List ports in use (Linux/Mac)
lsof -i :8080

# Kill process
kill -9 <PID>
```

### Database issues

```bash
# Reset database
docker-compose down -v
docker-compose up postgres -d
make migrate
```

## Contributing

1. Create feature branch: `git checkout -b feature/your-feature`
2. Run tests: `make test`
3. Run linters: `make lint`
4. Submit PR with description

All code must:
- Pass linters (golangci-lint, gosec)
- Have ≥80% test coverage
- Follow project conventions
- Include meaningful commit messages

## Documentation

- **Architecture:** See `ecommerce-platform-spec.md` (v1.0.0)
- **API Docs:** Auto-generated via Swagger (link: `/docs`)
- **Runbooks:** `docs/runbooks/`
- **Diagrams:** `docs/diagrams/` (PlantUML)

## Next Steps (Phase 2-5)

- Phase 2: Product catalog & search
- Phase 3: Commerce core (cart, checkout, payments)
- Phase 4: Observability & optimization
- Phase 5: Production readiness & scale testing

## License

© 2026 E-Commerce Platform. All rights reserved.

---

**Status:** Phase 1 Complete ✅  
**Date:** March 6, 2026  
**Next Phase:** Product & Catalog (Phase 2)
