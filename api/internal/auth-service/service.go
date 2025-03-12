package authservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vit0rr/chat/pkg/database/repositories"
	"github.com/vit0rr/chat/pkg/deps"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	deps  *deps.Deps
	Mongo *mongo.Database
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
}

type DeleteUserRequest struct {
	UserID string `json:"user_id"`
}

func NewService(deps *deps.Deps, db *mongo.Database) *Service {
	return &Service{
		deps:  deps,
		Mongo: db,
	}
}

// Register creates a new user account
func (s *Service) Register(ctx context.Context, b io.ReadCloser) (interface{}, error) {
	var req RegisterRequest
	err := json.NewDecoder(b).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %v", err)
	}
	defer b.Close()

	// Validate input
	if req.Email == "" || req.Password == "" || req.Nickname == "" {
		return nil, fmt.Errorf("email, password, and nickname are required")
	}

	// Check if user already exists
	existingUser, err := repositories.GetUserByEmail(ctx, s.Mongo, req.Email)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("failed to check existing user: %v", err)
	}

	if existingUser != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	// Create user
	newUser, err := repositories.CreateUser(ctx, s.Mongo, repositories.CreateUserData{
		Email:    req.Email,
		Password: string(hashedPassword),
		Nickname: req.Nickname,
		Activity: "offline",
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	// Generate JWT token
	userID := newUser.InsertedID.(string)
	token, err := generateJWT(userID, req.Email, req.Nickname, s.deps.Config.JWT.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %v", err)
	}

	return AuthResponse{
		Token:    token,
		UserID:   userID,
		Nickname: req.Nickname,
	}, nil
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(ctx context.Context, b io.ReadCloser) (interface{}, error) {
	var req LoginRequest
	err := json.NewDecoder(b).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %v", err)
	}
	defer b.Close()

	// Validate input
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	// Find user by email
	user, err := repositories.GetUserByEmail(ctx, s.Mongo, req.Email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Generate JWT token
	token, err := generateJWT(user.Id, user.Email, user.Nickname, s.deps.Config.JWT.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %v", err)
	}

	// Update user activity
	repositories.UpdateUser(ctx, s.Mongo, repositories.UpdateUserData{
		UserID:   user.Id,
		Activity: &[]string{"online"}[0],
	})

	return AuthResponse{
		Token:    token,
		UserID:   user.Id,
		Nickname: user.Nickname,
	}, nil
}

// DeleteUser deletes a user account
func (s *Service) DeleteUser(ctx context.Context, b io.ReadCloser) (interface{}, error) {
	var req DeleteUserRequest
	err := json.NewDecoder(b).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %v", err)
	}
	defer b.Close()

	// Validate input
	if req.UserID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Delete user
	err = repositories.DeleteUser(ctx, s.Mongo, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete user: %v", err)
	}

	return map[string]string{"message": "User deleted successfully"}, nil
}

// Helper function to generate JWT token
func generateJWT(userID, email, nickname, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      userID,
		"email":    email,
		"nickname": nickname,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
