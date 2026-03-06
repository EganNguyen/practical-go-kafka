package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/practical-go-kafka/shared/pkg/jwt"
)

// RequestIDMiddleware adds a request ID to each request for tracing
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// JWTAuthMiddleware validates JWT tokens on protected routes
func JWTAuthMiddleware(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := jwtManager.VerifyToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Add user context headers for downstream services
		c.Request.Header.Set("X-User-ID", claims.UserID)
		c.Request.Header.Set("X-User-Email", claims.Email)
		c.Set("user_id", claims.UserID)
		c.Set("roles", claims.Roles)
		c.Next()
	}
}

// RateLimitMiddleware implements token bucket rate limiting
type RateLimiter struct {
	limits map[string]*bucket
	mu     sync.RWMutex
}

type bucket struct {
	tokens    float64
	lastTime  time.Time
	capacity  float64
	refillRate float64
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*bucket),
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ip string, requestsPerMinute int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.limits[ip]
	now := time.Now()

	if !exists {
		refillRate := float64(requestsPerMinute) / 60.0 // tokens per second
		b = &bucket{
			tokens:     float64(requestsPerMinute),
			capacity:   float64(requestsPerMinute),
			refillRate: refillRate,
			lastTime:   now,
		}
		rl.limits[ip] = b
	} else {
		// Refill tokens
		elapsed := now.Sub(b.lastTime).Seconds()
		tokensToAdd := elapsed * b.refillRate
		b.tokens = min(b.tokens+tokensToAdd, b.capacity)
		b.lastTime = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// RateLimitMiddlewareFunc returns a Gin middleware function for rate limiting
func RateLimitMiddlewareFunc(limiter *RateLimiter, requestsPerMinute int) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !limiter.Allow(clientIP, requestsPerMinute) {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// CORSMiddleware handles CORS policy
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
