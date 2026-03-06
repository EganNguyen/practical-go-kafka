package jwt

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT claims for the platform
type Claims struct {
	UserID string   `json:"sub"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	JTI    string   `json:"jti"` // JWT ID for revocation tracking
	jwt.RegisteredClaims
}

// Manager handles JWT signing and verification
type Manager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// NewManager creates a new JWT manager
func NewManager(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) *Manager {
	return &Manager{
		privateKey: privateKey,
		publicKey:  publicKey,
	}
}

// GenerateAccessToken creates a new access token
func (m *Manager) GenerateAccessToken(userID, email string, roles []string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		JTI:    uuid.New().String(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// GenerateRefreshToken creates a new refresh token
func (m *Manager) GenerateRefreshToken(userID string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := &Claims{
		UserID: userID,
		JTI:    uuid.New().String(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// VerifyToken verifies and parses a JWT token
func (m *Manager) VerifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// ExtractClaims safely extracts claims from a token
func (m *Manager) ExtractClaims(tokenString string) (*Claims, error) {
	return m.VerifyToken(tokenString)
}
