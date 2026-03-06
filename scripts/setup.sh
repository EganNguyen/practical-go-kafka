#!/bin/bash

# Local Development Setup Script
# Initializes the environment for Phase 1 development

set -e

echo "🚀 E-Commerce Platform - Phase 1 Setup"
echo "========================================"
echo ""

# Check prerequisites
echo "✓ Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose."
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Go 1.22+ is not installed. Please install Go."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "  ✓ Docker: $(docker --version)"
echo "  ✓ Docker Compose: $(docker-compose --version)"
echo "  ✓ Go: $GO_VERSION"

# Create .env file if it doesn't exist
echo ""
echo "✓ Setting up environment variables..."

if [ ! -f .env ]; then
    cat > .env << EOF
# Database
DATABASE_URL=postgres://ecommerce:ecommerce-dev@localhost:5432/ecommerce
REDIS_URL=redis://localhost:6379
MONGO_URL=mongodb://ecommerce:ecommerce-dev@localhost:27017

# Kafka
KAFKA_BROKERS=localhost:9092

# Services
USER_SERVICE_PORT=8081
API_GATEWAY_PORT=8080
PRODUCT_SERVICE_PORT=8082
CART_SERVICE_PORT=8084
ORDER_SERVICE_PORT=8085
PAYMENT_SERVICE_PORT=8086
SEARCH_SERVICE_PORT=8088

# JWT (Development keys - change in production!)
JWT_PRIVATE_KEY_PATH=./etc/secrets/jwt_private_key
JWT_PUBLIC_KEY_PATH=./etc/secrets/jwt_public_key

# Environment
ENVIRONMENT=development
LOG_LEVEL=debug
EOF
    echo "  .env file created"
else
    echo "  .env file already exists"
fi

# Generate JWT keys if they don't exist
echo ""
echo "✓ Generating JWT keys..."

mkdir -p etc/secrets

if [ ! -f etc/secrets/jwt_private_key ] || [ ! -f etc/secrets/jwt_public_key ]; then
    # Generate 2048-bit RSA key pair
    openssl genrsa -out etc/secrets/jwt_private_key 2048 2>/dev/null
    openssl rsa -in etc/secrets/jwt_private_key -pubout -out etc/secrets/jwt_public_key 2>/dev/null
    echo "  ✓ JWT key pair generated"
else
    echo "  ✓ JWT keys already exist"
fi

# Download Go dependencies
echo ""
echo "✓ Downloading Go dependencies..."

cd shared && go mod download >/dev/null 2>&1 && cd ..
cd shared/pkg && go mod download >/dev/null 2>&1 && cd ../..
cd services/user-service && go mod download >/dev/null 2>&1 && cd ../..
cd services/api-gateway && go mod download >/dev/null 2>&1 && cd ../..

echo "  ✓ Dependencies downloaded"

# Install development tools
echo ""
echo "✓ Installing development tools..."

if ! command -v air &> /dev/null; then
    go install github.com/cosmtrek/air@latest
    echo "  ✓ air (hot reload) installed"
fi

if ! command -v golangci-lint &> /dev/null; then
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    echo "  ✓ golangci-lint installed"
fi

if ! command -v gosec &> /dev/null; then
    go install github.com/securego/gosec/v2/cmd/gosec@latest
    echo "  ✓ gosec installed"
fi

# Start infrastructure
echo ""
echo "✓ Starting Docker Compose services..."
docker-compose up -d

# Wait for services
echo ""
echo "✓ Waiting for services to be ready..."
sleep 15

# Check service health
echo ""
echo "✓ Checking service health..."

for service in postgres redis kafka elasticsearch; do
    if docker-compose ps | grep -q "$service.*Up"; then
        echo "  ✓ $service is running"
    else
        echo "  ⚠ $service may not be healthy yet, check logs with 'make logs'"
    fi
done

# Run migrations
echo ""
echo "✓ Running database migrations..."
docker-compose exec -T postgres psql -U ecommerce -d ecommerce -a \
    -f /docker-entrypoint-initdb.d/001_init_users.sql > /dev/null 2>&1
docker-compose exec -T postgres psql -U ecommerce -d ecommerce -a \
    -f /docker-entrypoint-initdb.d/002_init_products_inventory.sql > /dev/null 2>&1
docker-compose exec -T postgres psql -U ecommerce -d ecommerce -a \
    -f /docker-entrypoint-initdb.d/003_init_orders_payments.sql > /dev/null 2>&1

echo "  ✓ Migrations complete"

# Summary
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Setup Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🚀 Next Steps:"
echo ""
echo "  1. Start the services:"
echo "     make dev"
echo ""
echo "  2. Check logs:"
echo "     make logs"
echo ""
echo "  3. Test the API Gateway:"
echo "     curl http://localhost:8080/health"
echo ""
echo "  4. Register a user:"
echo "     curl -X POST http://localhost:8080/v1/auth/register \\"
echo "       -H 'Content-Type: application/json' \\"
echo "       -d '{\"email\":\"test@example.com\",\"password\":\"Test123456\",\"first_name\":\"Test\",\"last_name\":\"User\"}'"
echo ""
echo "📚 View API docs: http://localhost:8080/docs"
echo "📊 Database: psql -h localhost -U ecommerce -d ecommerce"
echo "🔍 Logs: make logs"
echo ""
echo "Happy coding! 🎉"
