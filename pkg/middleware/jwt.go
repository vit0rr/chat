package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vit0rr/chat/pkg/deps"
)

type contextKey string

const UserContextKey contextKey = "user"

type UserClaims struct {
	UserID   string
	Email    string
	Nickname string
}

func JWTAuth(deps *deps.Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Extract the token
			tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
			if tokenString == "" {
				http.Error(w, "Invalid token format", http.StatusUnauthorized)
				return
			}

			// Parse and validate the token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(deps.Config.JWT.Secret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			// Create user context
			userClaims := UserClaims{
				UserID:   claims["sub"].(string),
				Email:    claims["email"].(string),
				Nickname: claims["nickname"].(string),
			}

			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, userClaims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func isPublicPath(path string) bool {
	publicPaths := []string{
		"/api/v1/auth/register",
		"/api/v1/auth/login",
		"/swagger",
		"/",
	}

	for _, pp := range publicPaths {
		if strings.HasPrefix(path, pp) {
			return true
		}
	}

	return false
}
