# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY services/api-gateway/go.mod .
COPY services/api-gateway/go.sum .

# Download dependencies
RUN go mod download

# Copy source code
COPY services/api-gateway/ .
COPY shared/ ../shared

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api-gateway ./cmd/server

# Final stage
FROM alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/api-gateway .

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
	CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

EXPOSE 8080

CMD ["./api-gateway"]
