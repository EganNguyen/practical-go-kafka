# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY services/user-service/go.mod .
COPY services/user-service/go.sum .

# Download dependencies
RUN go mod download

# Copy source code
COPY services/user-service/ .
COPY shared/ ../shared

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/user-service ./cmd/server

# Final stage
FROM alpine:3.18

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/user-service .

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
	CMD ["/root/user-service", "--health"]

EXPOSE 8081

CMD ["./user-service"]
