package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Proxy handles forwarding requests to downstream services
type Proxy struct {
	userServiceURL    string
	productServiceURL string
	cartServiceURL    string
	orderServiceURL   string
	paymentServiceURL string
	searchServiceURL  string
}

// NewProxy creates a new proxy handler
func NewProxy(
	userServiceURL,
	productServiceURL,
	cartServiceURL,
	orderServiceURL,
	paymentServiceURL,
	searchServiceURL string,
) *Proxy {
	return &Proxy{
		userServiceURL:    userServiceURL,
		productServiceURL: productServiceURL,
		cartServiceURL:    cartServiceURL,
		orderServiceURL:   orderServiceURL,
		paymentServiceURL: paymentServiceURL,
		searchServiceURL:  searchServiceURL,
	}
}

// forwardRequest forwards a request to a downstream service
func (p *Proxy) forwardRequest(c *gin.Context, serviceURL string) {
	// Build downstream URL
	downstreamURL := serviceURL + c.Request.RequestURI

	// Create new request to downstream service
	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, downstreamURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to forward request"})
		return
	}

	// Copy headers
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Execute request
	client := &http.Client{
		Timeout: http.DefaultClient.Timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("downstream service error: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Copy response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to read response"})
		return
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
}

// Health returns the health status of the gateway
func (p *Proxy) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "api-gateway",
	})
}
