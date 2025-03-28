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

// @summary Register New User
// @description Creates a new user account with email, password, and nickname
// @tags auth
// @router /api/v1/auth/register [post]
// @param body body RegisterRequest true "User registration information"
// @produce application/json
// @success 201 {object} AuthResponse "User successfully registered with authentication token"
// @failure 400 {object} error "Bad request - Missing required fields or invalid input"
// @failure 409 {object} error "Conflict - User with this email already exists"
// @failure 500 {object} error "Internal server error"
func (s *Service) Register(ctx context.Context, b io.ReadCloser) (interface{}, error) {
	var req RegisterRequest
	err := json.NewDecoder(b).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %v", err)
	}
	defer b.Close()

	if req.Email == "" || req.Password == "" || req.Nickname == "" {
		return nil, fmt.Errorf("email, password, and nickname are required")
	}

	existingUser, err := repositories.GetUserByEmail(ctx, s.Mongo, req.Email)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, fmt.Errorf("failed to check existing user: %v", err)
	}

	if existingUser != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	newUser, err := repositories.CreateUser(ctx, s.Mongo, repositories.CreateUserData{
		Email:    req.Email,
		Password: string(hashedPassword),
		Nickname: req.Nickname,
		Activity: "offline",
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

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

// @summary User Login
// @description Authenticates a user with email and password, returning a JWT token
// @tags auth
// @router /api/v1/auth/login [post]
// @param body body LoginRequest true "User login credentials"
// @produce application/json
// @success 200 {object} AuthResponse "User successfully authenticated with token"
// @failure 400 {object} error "Bad request - Missing required fields"
// @failure 401 {object} error "Unauthorized - Invalid email or password"
// @failure 500 {object} error "Internal server error"
func (s *Service) Login(ctx context.Context, b io.ReadCloser) (interface{}, error) {
	var req LoginRequest
	err := json.NewDecoder(b).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %v", err)
	}
	defer b.Close()

	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	user, err := repositories.GetUserByEmail(ctx, s.Mongo, req.Email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	token, err := generateJWT(user.Id, user.Email, user.Nickname, s.deps.Config.JWT.Secret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %v", err)
	}

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

// @summary Delete User Account
// @description Permanently removes a user account and all associated data
// @tags auth
// @router /api/v1/auth/user [delete]
// @param body body DeleteUserRequest true "User ID to delete"
// @produce application/json
// @security JWT
// @success 200 {object} map[string]string "User successfully deleted"
// @failure 400 {object} error "Bad request - Missing user ID"
// @failure 401 {object} error "Unauthorized - Missing or invalid authentication"
// @failure 403 {object} error "Forbidden - Not authorized to delete this user"
// @failure 404 {object} error "Not found - User doesn't exist"
// @failure 500 {object} error "Internal server error"
func (s *Service) DeleteUser(ctx context.Context, b io.ReadCloser) (interface{}, error) {
	var req DeleteUserRequest
	err := json.NewDecoder(b).Decode(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %v", err)
	}
	defer b.Close()

	if req.UserID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	err = repositories.DeleteUser(ctx, s.Mongo, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete user: %v", err)
	}

	return map[string]string{"message": "User deleted successfully"}, nil
}

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
