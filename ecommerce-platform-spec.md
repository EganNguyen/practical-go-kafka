# Event-Driven Microservices E-Commerce Platform
### System Design Specification · v1.0.0 · March 2026

> **Status:** APPROVED &nbsp;|&nbsp; **Audience:** Engineering · Architecture · DevOps  
> **Stack:** Go · Kafka · Kubernetes · AWS &nbsp;|&nbsp; 10 Microservices · 1 Frontend

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [System Architecture](#2-system-architecture)
3. [Service Specifications](#3-service-specifications)
4. [Kafka Event Schema](#4-kafka-event-schema)
5. [Frontend — Storefront UI](#5-frontend--storefront-ui)
6. [Non-Functional Requirements](#6-non-functional-requirements)
7. [Infrastructure & DevOps](#7-infrastructure--devops)
8. [Testing Strategy](#8-testing-strategy)
9. [Repository & Project Structure](#9-repository--project-structure)
10. [Implementation Roadmap](#10-implementation-roadmap)

---

## 1. Project Overview

This document is the authoritative specification for the Event-Driven Microservices E-Commerce Platform. It defines architecture, service contracts, data models, API contracts, event schemas, and non-functional requirements. **All implementation MUST conform to this spec.**

### 1.1 Vision & Goals

- Handle millions of concurrent users in production without degradation
- Achieve sub-50 ms p99 latency on core read paths via CQRS + Redis caching
- Guarantee at-least-once event delivery and eventual consistency across services
- Enable independent deployment of every microservice with zero downtime
- Maintain 99.99% SLA on checkout and payment flows

### 1.2 Platform Summary

| Attribute | Value |
|---|---|
| **Language** | Go 1.22+ (all services) |
| **REST Framework** | Gin (primary), Gorilla Mux (gateway) |
| **Event Bus** | Apache Kafka 3.x — partitioned topics per domain |
| **Primary DB** | PostgreSQL 16 — transactional, relational data |
| **Document DB** | MongoDB 7 — catalog, reviews, content |
| **Cache / Queue** | Redis 7 — sessions, cart, rate-limit, pub/sub |
| **Container Runtime** | Docker + Kubernetes (EKS) + Helm charts |
| **IaC** | Terraform (AWS VPC, EKS, RDS, MSK, ElastiCache) |
| **CI/CD** | GitHub Actions — lint, test, build, push, deploy |
| **Observability** | Prometheus + Grafana + structured JSON logging (zerolog) |
| **Services** | 10 backend microservices + 1 Next.js frontend |

---

## 2. System Architecture

### 2.1 High-Level Architecture

The platform adopts a **CQRS + Event Sourcing** pattern at the Order domain level. All inter-service communication for asynchronous workflows uses Kafka. Synchronous calls (auth token validation, price lookups) use gRPC internally and REST externally. The API Gateway is the single external ingress point.

### 2.2 Service Topology

| # | Service | Port | Responsibility |
|---|---|---|---|
| 1 | `api-gateway` | 8080 | Single ingress — routing, auth middleware, rate-limit, SSL termination |
| 2 | `user-service` | 8081 | Registration, login, JWT issuance, profile management |
| 3 | `product-service` | 8082 | Catalog CRUD, search, categories, image metadata (MongoDB) |
| 4 | `inventory-service` | 8083 | Stock levels, reservations, adjustments — PostgreSQL |
| 5 | `cart-service` | 8084 | Session-based cart operations — Redis primary store |
| 6 | `order-service` | 8085 | Order lifecycle (CQRS write side) — Kafka producer |
| 7 | `payment-service` | 8086 | Stripe/PayPal integration, idempotency, refunds |
| 8 | `notification-service` | 8087 | Email/SMS/push via SendGrid & Twilio — Kafka consumer |
| 9 | `search-service` | 8088 | Elasticsearch-backed product search & autocomplete |
| 10 | `analytics-service` | 8089 | Event consumption, aggregations, reporting (read-only) |
| FE | `storefront-ui` | 3000 | Next.js 14 — SSR, React, Tailwind — customer-facing SPA |

### 2.3 Network Zones

- **Public Zone:** API Gateway only — exposed via AWS ALB + Route 53
- **Private Zone:** All microservices — reachable only within VPC / Kubernetes ClusterIP
- **Data Zone:** RDS PostgreSQL, MSK (Kafka), ElastiCache (Redis), DocumentDB — no direct internet access
- **Egress:** NAT Gateway for services calling external APIs (Stripe, SendGrid, Twilio)

### 2.4 Data Flow — Happy Path Checkout

| Step | Actor | Action |
|---|---|---|
| 1 | Client → API Gateway | `POST /v1/orders` — JWT validated, request forwarded |
| 2 | API Gateway → Order Service | Forward authenticated request with user context header |
| 3 | Order Service | Validate cart, lock inventory via Inventory Service (gRPC) |
| 4 | Order Service | Persist order (`PENDING`) to PostgreSQL — write side |
| 5 | Order Service → Kafka | Emit `order.created` event to `orders` topic |
| 6 | Payment Service | Consume `order.created` — charge via Stripe — emit `payment.completed` |
| 7 | Order Service | Consume `payment.completed` — update order to `CONFIRMED` |
| 8 | Inventory Service | Consume `order.confirmed` — decrement stock permanently |
| 9 | Notification Service | Consume `order.confirmed` — send email + push notification |
| 10 | Analytics Service | Consume all events — update materialized views |

---

## 3. Service Specifications

### 3.1 API Gateway (`api-gateway`)

> **Purpose:** Single entry point for all external traffic. Handles auth, routing, rate-limiting, and request tracing.

**Responsibilities:**
- JWT validation on all protected routes (RS256, JWKS endpoint)
- Route forwarding to downstream services via reverse proxy
- Rate limiting: 1000 req/min per IP (sliding window in Redis)
- Request ID injection (`X-Request-ID` header) for distributed tracing
- CORS policy enforcement and TLS termination

**Key Endpoints:**

| Method | Path | Forwards To |
|---|---|---|
| `ANY` | `/v1/auth/*` | user-service — no auth required |
| `ANY` | `/v1/products/*` | product-service — public GET, auth required for admin |
| `ANY` | `/v1/cart/*` | cart-service — auth required |
| `ANY` | `/v1/orders/*` | order-service — auth required |
| `ANY` | `/v1/payments/*` | payment-service — auth required |
| `ANY` | `/v1/search/*` | search-service — public |
| `GET` | `/v1/health` | Gateway health — no forwarding |

---

### 3.2 User Service (`user-service`)

> **Purpose:** Manages user identity, authentication, JWT lifecycle, and profile data. Source of truth for user records.

**Data Stores:**
- PostgreSQL — `users`, `roles`, `refresh_tokens` tables
- Redis — active session tokens (TTL: 15 min access / 7 day refresh)

**API Contracts:**

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| `POST` | `/v1/auth/register` | None | Create account, hash password (bcrypt cost 12) |
| `POST` | `/v1/auth/login` | None | Validate credentials, return JWT pair |
| `POST` | `/v1/auth/refresh` | Refresh token | Rotate access + refresh tokens |
| `POST` | `/v1/auth/logout` | Bearer JWT | Revoke refresh token, blacklist in Redis |
| `GET` | `/v1/users/me` | Bearer JWT | Return authenticated user profile |
| `PATCH` | `/v1/users/me` | Bearer JWT | Update profile fields |
| `DELETE` | `/v1/users/me` | Bearer JWT | Soft-delete account |

**JWT Specification:**
- Algorithm: RS256 — private key signs, public JWKS endpoint for verification
- Access token TTL: 15 minutes | Refresh token TTL: 7 days
- Claims: `sub` (user_id), `email`, `roles[]`, `iat`, `exp`, `jti`
- Token blacklist stored in Redis with TTL matching token expiry

---

### 3.3 Product Service (`product-service`)

> **Purpose:** Manages the product catalog, categories, pricing, and media references. Uses MongoDB for flexible schema.

**MongoDB Schema — `products` collection:**

| Field | Type | Notes |
|---|---|---|
| `_id` | ObjectID | Primary key |
| `sku` | string | Unique — indexed |
| `name` | string | Full-text indexed |
| `description` | string | Markdown supported |
| `price` | decimal128 | Stored as cents to avoid float rounding |
| `category_ids` | `[]ObjectID` | References `categories` collection |
| `images` | `[]ImageRef` | CDN URLs + dimensions |
| `attributes` | `map[string]any` | Size, color, material — flexible |
| `status` | enum | `active` \| `inactive` \| `draft` |
| `created_at` / `updated_at` | time.Time | RFC3339 UTC |

---

### 3.4 Inventory Service (`inventory-service`)

> **Purpose:** Tracks stock levels and handles transactional reservations to prevent overselling.

**PostgreSQL Schema — `inventory` table:**

| Column | Type | Constraint / Notes |
|---|---|---|
| `id` | UUID | PK |
| `sku` | VARCHAR(64) | UNIQUE NOT NULL — FK to product |
| `quantity_available` | INTEGER | `CHECK >= 0` — enforced at DB level |
| `quantity_reserved` | INTEGER | Active reservation hold |
| `warehouse_id` | UUID | Multi-warehouse support |
| `version` | BIGINT | Optimistic locking |
| `updated_at` | TIMESTAMPTZ | Auto-updated trigger |

**Reservation Protocol:**
- **Reserve:** `BEGIN TX` → `SELECT FOR UPDATE` → check available ≥ qty → decrement available, increment reserved → `COMMIT`
- **Confirm:** On `order.confirmed` event — decrement reserved permanently
- **Release:** On `order.cancelled` / `payment.failed` — restore reserved back to available
- **Timeout:** Reservations older than 10 minutes auto-released by background cron

---

### 3.5 Cart Service (`cart-service`)

> **Purpose:** Manages user shopping carts with Redis as the primary store for low-latency access.

**Redis Data Model:**
- Key pattern: `cart:{user_id}` — Redis Hash
- Field: `{sku}` → JSON `{qty, price_snapshot, added_at}`
- TTL: 30 days — refreshed on every write operation
- Guest cart key: `cart:guest:{session_id}` — merged on login

**API Contracts:**

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/v1/cart` | Return all cart items with current prices |
| `PUT` | `/v1/cart/items` | Add or update item (upsert by SKU) |
| `DELETE` | `/v1/cart/items/:sku` | Remove single item |
| `DELETE` | `/v1/cart` | Clear entire cart |
| `POST` | `/v1/cart/checkout-preview` | Validate stock, apply promotions, return total |

---

### 3.6 Order Service (`order-service`) — CQRS

> **Purpose:** Write side of CQRS. Creates orders, manages state machine, and publishes domain events to Kafka.

**Order State Machine:**

| From State | To State | Trigger |
|---|---|---|
| — | `PENDING` | Order created, inventory reserved |
| `PENDING` | `AWAITING_PAYMENT` | Checkout confirmed by user |
| `AWAITING_PAYMENT` | `CONFIRMED` | `payment.completed` event received |
| `AWAITING_PAYMENT` | `CANCELLED` | `payment.failed` or timeout (30 min) |
| `CONFIRMED` | `PROCESSING` | Fulfillment started |
| `PROCESSING` | `SHIPPED` | Shipping label generated |
| `SHIPPED` | `DELIVERED` | Delivery confirmed |
| `CONFIRMED` / `PROCESSING` | `REFUND_REQUESTED` | Customer initiated |
| `REFUND_REQUESTED` | `REFUNDED` | `payment.refunded` event received |

**Idempotency:**
- All order creation requests MUST include `Idempotency-Key` header (UUID v4)
- Keys stored in Redis with 24h TTL — duplicate requests return cached response
- Database unique constraint on `idempotency_key` column as secondary guard

---

### 3.7 Payment Service (`payment-service`)

> **Purpose:** Integrates with Stripe and PayPal. Handles charge, capture, refund, and webhook verification.

**Payment Flow:**
1. Consume `order.awaiting_payment` from Kafka
2. Create Stripe `PaymentIntent` with order amount and metadata
3. Return `client_secret` to frontend via order-service callback
4. Frontend confirms payment — Stripe webhook notifies payment-service
5. Verify webhook signature (`Stripe-Signature` header — HMAC SHA256)
6. Emit `payment.completed` or `payment.failed` to Kafka

**Idempotency & Safety:**
- All Stripe calls use idempotency keys: `payment:{order_id}:{attempt_number}`
- Webhook events stored in `payments_events` table — deduplicated by `stripe_event_id`
- Refund flow: `refund.initiated` → Stripe refund → `payment.refunded` event
- PCI compliance: **NO card data stored**; Stripe handles tokenization entirely

---

### 3.8 Notification Service (`notification-service`)

> **Purpose:** Consumes domain events and dispatches email, SMS, and push notifications.

**Event Subscriptions:**

| Kafka Topic / Event | Channel | Template |
|---|---|---|
| `orders` — `order.confirmed` | Email + Push | Order confirmation with items & total |
| `orders` — `order.shipped` | Email + SMS | Shipping notification with tracking link |
| `orders` — `order.delivered` | Email + Push | Delivery confirmation + review prompt |
| `orders` — `order.cancelled` | Email | Cancellation notice with refund timeline |
| `payments` — `payment.failed` | Email + Push | Payment failure with retry CTA |
| `users` — `user.registered` | Email | Welcome email with verification link |
| `users` — `password.reset` | Email | Secure reset link (10 min TTL) |

---

### 3.9 Search Service (`search-service`)

> **Purpose:** Elasticsearch-backed service providing full-text search, faceted filtering, and autocomplete for products.

**Elasticsearch Index — `products`:**
- **Mappings:** `name` (text, analyzed), `description` (text), `sku` (keyword), `category_ids` (keyword), `price` (scaled_float), `status` (keyword), `attributes` (flattened)
- **Analyzer:** custom `edge_ngram` for autocomplete (min=2, max=15)
- **Index updated** via Kafka consumer — `product.created` / `product.updated` events
- Stale reads acceptable — eventual consistency within 2 seconds

**API:**

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/v1/search?q=&category=&min_price=&max_price=&page=` | Full-text search with filters + pagination |
| `GET` | `/v1/search/autocomplete?q=` | Returns top 10 suggestions |
| `GET` | `/v1/search/facets?q=` | Returns aggregation buckets for filter UI |

---

### 3.10 Analytics Service (`analytics-service`)

> **Purpose:** Read-only event consumer. Builds materialized views for dashboards and reporting. No external writes.

**Consumed Topics:**
- `orders` — all order events for funnel analysis and GMV reporting
- `payments` — revenue metrics, refund rates
- `users` — registration trends, DAU/MAU
- `inventory` — stock velocity, reorder signals

**Materialized Views (PostgreSQL):**
- `daily_revenue`: `date`, `total_gmv`, `order_count`, `avg_order_value`
- `product_sales`: `sku`, `units_sold_30d`, `revenue_30d`, `return_rate`
- `user_cohorts`: `signup_week`, `orders_count`, `ltv`, `churn_indicator`
- Views refreshed by background worker every 5 minutes — Prometheus gauge for lag

---

## 4. Kafka Event Schema

### 4.1 Topic Naming Convention

All topics follow the pattern: `{domain}.{version}` — e.g. `orders.v1`, `payments.v1`, `users.v1`, `inventory.v1`, `products.v1`

### 4.2 Base Event Envelope

All events **MUST** conform to this envelope (JSON, UTF-8):

| Field | Type | Description |
|---|---|---|
| `event_id` | UUID v4 | Globally unique — used for deduplication |
| `event_type` | string | Domain-scoped: `order.created`, `payment.completed`, etc. |
| `aggregate_id` | UUID | ID of the entity (`order_id`, `user_id`, `sku`) |
| `aggregate_type` | string | `order` \| `payment` \| `user` \| `product` \| `inventory` |
| `version` | integer | Schema version — start at 1, increment on breaking changes |
| `timestamp` | RFC3339 | UTC — event emission time |
| `correlation_id` | UUID | Trace ID propagated from originating HTTP request |
| `producer_service` | string | Emitting service name (e.g. `order-service`) |
| `payload` | object | Event-specific data — see 4.3 |

### 4.3 Event Catalog

| Event Type | Producer | Key Payload Fields |
|---|---|---|
| `order.created` | order-service | `order_id`, `user_id`, `items[]`, `total_amount`, `currency` |
| `order.confirmed` | order-service | `order_id`, `payment_id`, `confirmed_at` |
| `order.shipped` | order-service | `order_id`, `tracking_number`, `carrier`, `estimated_delivery` |
| `order.delivered` | order-service | `order_id`, `delivered_at` |
| `order.cancelled` | order-service | `order_id`, `reason`, `cancelled_at` |
| `payment.completed` | payment-service | `payment_id`, `order_id`, `amount`, `provider`, `provider_txn_id` |
| `payment.failed` | payment-service | `payment_id`, `order_id`, `failure_code`, `failure_message` |
| `payment.refunded` | payment-service | `payment_id`, `order_id`, `refund_amount`, `refunded_at` |
| `user.registered` | user-service | `user_id`, `email`, `created_at` |
| `user.password_reset` | user-service | `user_id`, `reset_token_hash`, `expires_at` |
| `inventory.reserved` | inventory-service | `sku`, `qty_reserved`, `order_id` |
| `inventory.released` | inventory-service | `sku`, `qty_released`, `order_id`, `reason` |
| `product.created` | product-service | `sku`, `name`, `price`, `category_ids` |
| `product.updated` | product-service | `sku`, `changed_fields[]`, `updated_at` |

### 4.4 Consumer Group Strategy

- Each service has a dedicated consumer group: `{service-name}-cg`
- Dead Letter Topic: `{topic}.dlt` — events that fail after 3 retry attempts
- Retry backoff: `1s → 5s → 30s` (exponential) before sending to DLT
- Partition key: `aggregate_id` — ensures ordering per entity
- Replication factor: `3` | Min ISR: `2` | Retention: `7 days`

---

## 5. Frontend — Storefront UI

### 5.1 Technology Stack

| Technology | Usage |
|---|---|
| **Next.js 14** | App Router — SSR for product pages, ISR for catalog |
| **React 18** | Client components for cart, auth, checkout flows |
| **TypeScript** | Strict mode — full type safety across all components |
| **Tailwind CSS** | Utility-first styling — design system tokens via CSS variables |
| **React Query** | Server state management — API caching, optimistic updates |
| **Zustand** | Client state — cart count, user session, UI state |
| **Stripe.js** | PCI-compliant card element — zero card data in our code |
| **Axios** | HTTP client with interceptors for JWT refresh and error handling |
| **Vitest + Testing Library** | Unit and integration tests — 80% coverage target |
| **Playwright** | E2E tests — critical paths: browse, cart, checkout |

### 5.2 Page Architecture

| Route | Rendering | Description |
|---|---|---|
| `/` | SSG + ISR | Homepage — hero, featured products, promotions |
| `/products` | SSR | Catalog with search, filters, pagination |
| `/products/[slug]` | ISR (60s) | Product detail — images, specs, add to cart |
| `/cart` | CSR | Cart review — quantities, remove, subtotal |
| `/checkout` | CSR (auth) | Address, shipping, payment — multi-step wizard |
| `/checkout/success` | SSR | Order confirmation with order ID |
| `/account` | CSR (auth) | Profile, order history, addresses |
| `/account/orders/[id]` | SSR (auth) | Order detail + tracking |
| `/search` | SSR | Search results — calls search-service |
| `/admin/*` | CSR (admin role) | Product, order, user management |

### 5.3 Authentication Flow

- Access token stored **in memory** (Zustand) — never in `localStorage`
- Refresh token stored in `HttpOnly Secure` cookie — not accessible by JS
- Silent refresh: Axios interceptor catches `401`, calls `/v1/auth/refresh`, retries original request
- On logout: DELETE refresh token server-side, clear Zustand state and cookie

---

## 6. Non-Functional Requirements

### 6.1 Performance Targets

| Metric | Target p50 | Target p99 | SLA |
|---|---|---|---|
| Product page load (SSR) | 80 ms | 200 ms | 99.9% |
| Search query response | 30 ms | 100 ms | 99.9% |
| Cart GET (Redis) | 5 ms | 20 ms | 99.99% |
| Order creation (write) | 100 ms | 400 ms | 99.9% |
| Payment processing | 200 ms | 800 ms | 99.99% |
| Kafka event delivery | < 100 ms | < 500 ms | 99.999% |
| Overall platform uptime | — | — | **99.99%** |

### 6.2 Scalability

- **Horizontal scaling:** All services are stateless — scale via Kubernetes HPA
- HPA triggers: CPU > 70% or custom Kafka consumer lag metric > 10,000
- **Database:** PostgreSQL RDS Multi-AZ + read replicas for analytics queries
- **Kafka:** 12 partitions per topic — enables 12 parallel consumers per group
- **Redis:** ElastiCache cluster mode with 3 shards — auto-failover enabled

### 6.3 Security Requirements

- All external traffic over TLS 1.3 — no HTTP allowed
- JWT RS256 — private key stored in AWS Secrets Manager, rotated quarterly
- Secrets: All service credentials via Kubernetes Secrets (backed by AWS Secrets Manager)
- Input validation: All request bodies validated against JSON Schema before processing
- SQL injection: Parameterized queries only — ORM raw query usage prohibited
- Rate limiting: Per-IP at gateway, per-user-ID at service level for auth endpoints
- OWASP Top 10 compliance verified in CI via `gosec` static analysis

### 6.4 Resilience

- **Circuit breaker:** `gobreaker` — trip after 5 consecutive failures, reset after 60s
- **Timeout:** All outgoing HTTP calls have 5s timeout; gRPC calls 2s deadline
- **Retry:** Idempotent GET calls retry 3x with exponential backoff
- **Bulkhead:** Each service has separate connection pool to DB — prevents cascade failures
- **Graceful shutdown:** SIGTERM triggers 30s drain window — in-flight requests complete

---

## 7. Infrastructure & DevOps

### 7.1 AWS Resources (Terraform Managed)

| Resource | Service | Configuration |
|---|---|---|
| VPC | networking | 3 AZs, public + private + data subnets |
| EKS | kubernetes | v1.30, managed node groups, IRSA for service accounts |
| RDS PostgreSQL | databases | `db.r6g.xlarge`, Multi-AZ, automated backups 7 days |
| MSK (Kafka) | messaging | `kafka.m5.large` x3, 3 brokers, encryption at rest |
| ElastiCache Redis | caching | `cache.r6g.large`, cluster mode, 3 shards x 2 replicas |
| DocumentDB | mongodb | `db.r6g.large` x3, TLS, automated snapshots |
| ALB | load balancing | HTTPS listener, WAF attached, access logs to S3 |
| ECR | container registry | Private, image scanning on push, lifecycle policy |
| S3 | storage | Product images, Terraform state, ALB access logs |
| CloudWatch | monitoring | Log groups per service, custom metrics from Prometheus |
| Secrets Manager | secrets | JWT keys, DB passwords, API keys — auto-rotation |
| Route 53 | dns | Hosted zone, ALB alias records, health checks |

### 7.2 CI/CD Pipeline (GitHub Actions)

**Per-Service Workflow Stages:**

1. **Lint & Vet** — `golangci-lint`, `go vet`, `govulncheck` security scan
2. **Test** — `go test ./... -race -coverprofile` — fail if coverage < 80%
3. **Build** — Docker multi-stage build — distroless final image
4. **Push** — Tag with git SHA + branch + semver; push to ECR
5. **Deploy Staging** — `helm upgrade --install` with `--atomic` flag
6. **Smoke Test** — curl health endpoints; run Playwright E2E subset
7. **Deploy Production** — Manual approval gate → rolling update via Helm

### 7.3 Observability Stack

**Metrics — Prometheus + Grafana:**
- All services expose `/metrics` endpoint — standard Go runtime + custom business metrics
- Key metrics: `http_request_duration_seconds`, `kafka_consumer_lag`, `db_query_duration_seconds`, `order_created_total`, `payment_success_rate`
- Grafana dashboards: per-service overview, Kafka health, database connections, business KPIs
- Alerting: PagerDuty integration — p99 latency > 500ms, error rate > 1%, consumer lag > 50k

**Logging — zerolog (structured JSON):**
- Every log line includes: `timestamp`, `level`, `service`, `trace_id`, `span_id`, `user_id` (if auth), `message`, `error`
- Log levels: `DEBUG` (dev only), `INFO` (normal ops), `WARN` (recoverable), `ERROR` (investigate), `FATAL` (crash)
- Logs shipped to CloudWatch Logs → Elasticsearch for search and alerting

---

## 8. Testing Strategy

### 8.1 Coverage Requirements

| Test Type | Coverage Target | Tooling |
|---|---|---|
| Unit Tests | ≥ 80% | `go test`, `testify/assert`, `gomock` |
| Integration Tests | Core flows | `go test` + `testcontainers` (real DB/Redis/Kafka) |
| API Contract Tests | All endpoints | Postman Collections in CI |
| E2E Tests | Happy path + auth | Playwright — checkout, login, order history |
| Load Tests | 1M concurrent | `k6` — ramp up profile, p99 assertions |
| Security Scans | Every build | `gosec`, `govulncheck`, Trivy (image CVE scan) |

### 8.2 Unit Test Conventions

- Test files co-located with source: `foo.go` → `foo_test.go`
- Table-driven tests for all business logic functions
- Mock external dependencies via interfaces — `gomock` code generation
- No real network calls in unit tests — all external deps mocked

### 8.3 Integration Test Approach

- Use `testcontainers-go` to spin up PostgreSQL, Redis, Kafka per test suite
- Each test suite gets isolated schema/database — no shared state
- Test realistic Kafka event flows: produce event → consume → assert side effects
- Run in CI on every PR — parallelized across 4 runners

### 8.4 Load Test Scenarios

- **Browse scenario (70%):** product search + product detail pages
- **Purchase scenario (20%):** add to cart + checkout + payment
- **Admin scenario (10%):** product CRUD + order management
- **Ramp profile:** 0 → 100K VUs over 10 min, hold 30 min, ramp down 5 min
- **Pass criteria:** p99 < 500ms, error rate < 0.1%, zero data loss

---

## 9. Repository & Project Structure

### 9.1 Monorepo Layout

```
/
├── services/
│   ├── api-gateway/          # Gateway — main.go, handlers, middleware
│   ├── user-service/         # User auth service
│   ├── product-service/      # Product catalog service
│   ├── inventory-service/    # Stock management service
│   ├── cart-service/         # Cart management service
│   ├── order-service/        # CQRS order service
│   ├── payment-service/      # Payment integration service
│   ├── notification-service/ # Notifications consumer
│   ├── search-service/       # Elasticsearch search service
│   └── analytics-service/    # Analytics consumer service
├── frontend/
│   └── storefront-ui/        # Next.js 14 storefront application
├── shared/                   # Go modules shared across services (events, models, pkg)
├── infra/
│   ├── terraform/            # AWS infrastructure — modules per resource type
│   ├── helm/                 # Helm charts — one chart per service + umbrella chart
│   └── k8s/                  # Raw Kubernetes manifests (ConfigMaps, NetworkPolicies)
├── .github/
│   └── workflows/            # CI/CD pipelines — per-service and global
├── scripts/                  # Dev tooling — seed data, local setup, db migrations
└── docs/                     # Architecture diagrams (PlantUML), API specs (OpenAPI 3.1)
```

### 9.2 Per-Service Go Package Structure

```
{service}/
├── cmd/
│   └── server/
│       └── main.go           # Entry point, DI wiring, server bootstrap
├── internal/
│   ├── handler/              # HTTP handlers (thin — no business logic)
│   ├── service/              # Business logic layer — all core domain rules
│   ├── repository/           # Database access layer — interface + implementation
│   ├── middleware/           # Auth, logging, rate-limit, recovery middleware
│   ├── model/                # Domain structs, request/response DTOs
│   ├── events/               # Kafka producer and consumer implementations
│   └── config/               # Env-based config using viper
├── Dockerfile
├── Makefile
└── go.mod
```

### 9.3 Local Development

| Command | Action |
|---|---|
| `docker-compose up` | Starts PostgreSQL, MongoDB, Redis, Kafka, Zookeeper, Elasticsearch |
| `make dev` | Starts all services with hot-reload (`air`) |
| `make test` | Runs full unit + integration test suite |
| `make lint` | Runs `golangci-lint` with project ruleset |
| `make migrate` | Applies DB migrations using `golang-migrate` |

> Seed data script populates 10K products and 1K users for realistic local testing.

---

## 10. Implementation Roadmap

### Phase 1 — Core Foundation

- Monorepo structure, Terraform AWS setup (VPC, EKS, RDS, MSK, ElastiCache)
- CI/CD pipeline: lint → test → build → push to ECR
- Helm charts for all 11 services — umbrella chart for local dev
- User Service: registration, login, JWT RS256, refresh token rotation
- API Gateway: routing, JWT middleware, rate limiting, request ID injection
- Shared event envelope library — validate schema on publish and consume

### Phase 2 — Product & Catalog

- Product Service: CRUD, MongoDB schema, image upload to S3
- Inventory Service: stock management, reservation protocol, optimistic locking
- Search Service: Elasticsearch index setup, full-text search, autocomplete
- Frontend: product listing, product detail, search page (SSR/ISR)
- Product events → Search Service Kafka consumer (catalog sync pipeline)

### Phase 3 — Commerce Core

- Cart Service: Redis data model, add/remove/clear, guest → user merge on login
- Order Service: CQRS write side, state machine, idempotency, Kafka producer
- Payment Service: Stripe integration, webhook handling, idempotency keys
- Notification Service: Kafka consumers, SendGrid email, Twilio SMS
- Frontend: cart page, multi-step checkout wizard, Stripe.js payment element
- E2E test suite: full checkout happy path automated in Playwright

### Phase 4 — Observability & Scale

- Analytics Service: event consumers, materialized views, admin dashboard API
- Prometheus metrics in all services — Grafana dashboards and PagerDuty alerts
- Structured logging (zerolog) — `trace_id` propagation across all services
- Load testing with k6 — identify and resolve bottlenecks, tune HPA
- Security hardening: `gosec`, Trivy scans, pen-test critical paths
- Documentation: OpenAPI 3.1 specs, architecture diagrams, runbooks

### Phase 5 — Production Readiness

- Chaos engineering: simulate service failures, Kafka consumer lag, DB failover
- Blue/green deployment validation — zero-downtime rollout drill
- SLA validation: sustained 1M concurrent user load test for 1 hour
- Security review: final OWASP Top 10 audit, JWT rotation drill
- Go-live: DNS cutover, monitoring war-room, rollback runbook ready

---

*End of Specification — v1.0.0*
