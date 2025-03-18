package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vit0rr/chat/pkg/deps"
	"github.com/vit0rr/chat/pkg/log"
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

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// As I know, it's not possible to pass a header to a websocket, 
				// so I'm checking if the token is in the query params too
				tokenString := r.URL.Query().Get("token")
				if tokenString == "" {
					log.Error(r.Context(), "Authorization header required", log.ErrAttr(errors.New("authorization header required")))
					http.Error(w, "Authorization header required", http.StatusUnauthorized)
					return
				}

				authHeader = "Bearer " + tokenString
			}

			// Extract the token
			tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
			if tokenString == "" {
				log.Error(r.Context(), "Invalid token format", log.ErrAttr(errors.New("invalid token format")))
				http.Error(w, "Invalid token format", http.StatusUnauthorized)
				return
			}

			// Parse and validate the token with current secret
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(deps.Config.JWT.Secret), nil
			})

			// If token is invalid with current secret, return unauthorized error
			if err != nil || !token.Valid {
				log.Error(r.Context(), "Invalid or expired token", log.ErrAttr(errors.New("invalid or expired token")))
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

	if path == "/" {
		return true
	}
	

	if strings.HasPrefix(path, "/swagger") {
		return true
	}

	for _, pp := range publicPaths {
		if path == pp {
			return true
		}
	}

	return false
}
