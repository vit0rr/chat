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

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// As I know, it's not possible to pass a header to a websocket, 
				// so I'm checking if the token is in the query params too
				tokenString := r.URL.Query().Get("token")
				if tokenString == "" {
					http.Error(w, "Authorization header required", http.StatusUnauthorized)
					return
				}

				authHeader = "Bearer " + tokenString
			}

			// Extract the token
			tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
			if tokenString == "" {
				http.Error(w, "Invalid token format", http.StatusUnauthorized)
				return
			}

			// Parse and validate the token with current secret
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(deps.Config.JWT.Secret), nil
			})

			// If token is invalid with current secret, try with old secret
			if err != nil || !token.Valid {
				oldToken, oldErr := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					return []byte(deps.Config.JWT.OldSecret), nil
				})

				if oldErr != nil || !oldToken.Valid {
					http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
					return
				}

				// token is valid with old secret, issue a new token
				claims, ok := oldToken.Claims.(jwt.MapClaims)
				if !ok {
					http.Error(w, "Invalid token claims", http.StatusUnauthorized)
					return
				}

				userClaims := UserClaims{
					UserID:   claims["sub"].(string),
					Email:    claims["email"].(string),
					Nickname: claims["nickname"].(string),
				}

				newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"sub":      userClaims.UserID,
					"email":    userClaims.Email,
					"nickname": userClaims.Nickname,
					"exp":      claims["exp"], // Preserve original expiration
				})

				newTokenString, signErr := newToken.SignedString([]byte(deps.Config.JWT.Secret))
				if signErr != nil {
					http.Error(w, "Error creating new token", http.StatusInternalServerError)
					return
				}

				w.Header().Set("X-New-Token", newTokenString)

				ctx := context.WithValue(r.Context(), UserContextKey, userClaims)
				next.ServeHTTP(w, r.WithContext(ctx))
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
