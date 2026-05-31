package middleware

import (
	"log"
	"net/http"
	"time"
)

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

// CORSMiddleware adds necessary headers for cross-origin requests
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-Header, Authorization")

		// Handle preflight options requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
