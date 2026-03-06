package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/practical-go-kafka/shared/model"
	"github.com/practical-go-kafka/shared/pkg/jwt"
	"github.com/practical-go-kafka/user-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles business logic for user management
type UserService struct {
	repo       repository.UserRepository
	jwtManager *jwt.Manager
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewUserService creates a new user service
func NewUserService(
	repo repository.UserRepository,
	jwtManager *jwt.Manager,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *UserService {
	return &UserService{
		repo:       repo,
		jwtManager: jwtManager,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// Register creates a new user account
func (s *UserService) Register(ctx context.Context, req *model.RegisterRequest) (*model.TokenResponse, error) {
	// Hash password with bcrypt cost 12
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &model.User{
		ID:        uuid.New(),
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Status:    "active",
		Roles:     []string{"user"},
	}

	// Create user in repository
	if err := s.repo.Create(ctx, user, string(hashedPassword)); err != nil {
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	// Generate tokens
	return s.generateTokens(user.ID, user.Email, user.Roles)
}

// Login authenticates a user and returns JWT tokens
func (s *UserService) Login(ctx context.Context, req *model.LoginRequest) (*model.TokenResponse, error) {
	// Retrieve user by email
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate tokens
	return s.generateTokens(user.ID, user.Email, user.Roles)
}

// RefreshToken generates new access and refresh tokens
func (s *UserService) RefreshToken(ctx context.Context, oldRefreshToken string) (*model.TokenResponse, error) {
	// Verify refresh token
	claims, err := s.jwtManager.VerifyToken(oldRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Get user to verify they still exist
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if user.Status != "active" {
		return nil, fmt.Errorf("user account is not active")
	}

	// Generate new tokens
	return s.generateTokens(user.ID, user.Email, user.Roles)
}

// GetProfile retrieves user profile information
func (s *UserService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.UserProfileResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &model.UserProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Roles:     user.Roles,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// UpdateProfile updates user profile information
func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName string) (*model.UserProfileResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.FirstName = firstName
	user.LastName = lastName

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &model.UserProfileResponse{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Roles:     user.Roles,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// DeleteAccount soft-deletes a user account
func (s *UserService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	return s.repo.Delete(ctx, userID)
}

// generateTokens creates access and refresh JWT tokens
func (s *UserService) generateTokens(userID uuid.UUID, email string, roles []string) (*model.TokenResponse, error) {
	accessToken, err := s.jwtManager.GenerateAccessToken(userID.String(), email, roles, s.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	if len(roles) == 0 {
		roles = []string{"user"}
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(userID.String(), s.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &model.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTTL.Seconds()),
	}, nil
}
