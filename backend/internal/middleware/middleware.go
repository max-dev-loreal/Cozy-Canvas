package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "user"
const UserIDContextKey contextKey = "user_id"

// responseWriter captures the status code of the HTTP response
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs the request method, path, status, and duration
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		log.Printf("[API] %s %s -> %d %s (%v)", 
			r.Method, 
			r.URL.Path, 
			wrapped.status, 
			http.StatusText(wrapped.status), 
			time.Since(start),
		)
	})
}

// CORSMiddleware adds necessary headers for cross-origin requests and validates the Origin header against the CORS_ALLOWED_ORIGINS environment variable.
func CORSMiddleware(next http.Handler) http.Handler {
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	origins := strings.Split(allowedOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" {
			isAllowed := false
			for _, o := range origins {
				if o == "*" || o == origin {
					isAllowed = true
					break
				}
			}

			if !isAllowed {
				log.Printf("[CORS] Blocked request from unauthorized origin: %s", origin)
				http.Error(w, "Forbidden: CORS origin not allowed", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			// For non-CORS requests (like server-to-server or direct browser navigation), 
			// we can either allow or strictly require an origin. 
			// Usually, we allow them to pass through but don't set CORS headers.
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight options requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware validates the JWT token in the Authorization header
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		// Expected format: "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			http.Error(w, "JWT secret configuration missing on server", http.StatusInternalServerError)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			username, _ := claims["username"].(string)
			var userID int
			if uidVal, ok := claims["user_id"]; ok {
				switch v := uidVal.(type) {
				case float64:
					userID = int(v)
				case int:
					userID = v
				}
			}
			// Add username and user_id to context
			ctx := context.WithValue(r.Context(), UserContextKey, username)
			ctx = context.WithValue(ctx, UserIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		}
	})
}
