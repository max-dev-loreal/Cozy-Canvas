package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cozy-canvas/backend/internal/handlers"
	"cozy-canvas/backend/internal/middleware"
	"cozy-canvas/backend/internal/storage"
	"cozy-canvas/backend/internal/store"

	"github.com/go-chi/chi/v5"

	// Standard PostgreSQL driver for Go
	_ "github.com/lib/pq"
)

func main() {
	log.Println("[INFO] Starting Cozy Canvas REST API Backend...")

	// 12-Factor App: Configuration via Environment Variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Strict security check: JWT_SECRET must be configured
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("[FATAL] JWT_SECRET environment variable is not set. Refusing to start for security reasons.")
	}

	dbURL := os.Getenv("DATABASE_URL")
	var userRepo store.UserRepository
	var noteRepo store.NoteRepository
	var connRepo store.ConnectionRepository
	var db *sql.DB
	var err error

	if dbURL != "" {
		log.Printf("[DB] Connecting to PostgreSQL at %s...\n", dbURL)
		db, err = sql.Open("postgres", dbURL)
		if err == nil {
			err = db.Ping()
		}

		if err != nil {
			log.Printf("[ERROR] Database connection failed: %v. Falling back to MemoryStore (Not implemented for Repository Pattern yet).\n", err)
			// For simplicity in this refactor, we assume DB is available. 
			// In a real app, we'd have Memory implementations for each repo.
			log.Fatal("Database is required for current implementation")
		} else {
			log.Println("[DB] Successfully connected to PostgreSQL! Active production mode.")
			userRepo = store.NewPGUserRepository(db)
			noteRepo = store.NewPGNoteRepository(db)
			connRepo = store.NewPGConnectionRepository(db)
		}
	} else {
		log.Fatal("DATABASE_URL env variable not provided.")
	}

	// Read MinIO environment variables
	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	minioAccessKey := os.Getenv("MINIO_ACCESS_KEY")
	minioSecretKey := os.Getenv("MINIO_SECRET_KEY")
	minioBucket := os.Getenv("MINIO_BUCKET")
	minioSecure := false
	if os.Getenv("MINIO_USE_SSL") == "true" {
		minioSecure = true
	}

	var minioClient *storage.MinIOClient
	if minioEndpoint != "" && minioAccessKey != "" && minioSecretKey != "" && minioBucket != "" {
		log.Printf("[MinIO] Connecting to MinIO endpoint %s (bucket: %s, secure: %t)...\n", minioEndpoint, minioBucket, minioSecure)
		minioClient, err = storage.NewMinIOClient(minioEndpoint, minioAccessKey, minioSecretKey, minioBucket, minioSecure)
		if err != nil {
			log.Printf("[ERROR] MinIO initialization failed: %v\n", err)
		} else {
			log.Println("[MinIO] MinIO storage client successfully initialized.")
		}
	} else {
		log.Println("[WARN] MinIO environment variables not fully set. Storage features will be unavailable.")
	}

	// Initialize API handlers with specific repositories and storage client
	apiHandler := handlers.NewAPIHandler(userRepo, noteRepo, connRepo, minioClient)

	// Create Chi router for modern route handling
	r := chi.NewRouter()

	// Public routes
	r.Post("/api/auth/register", apiHandler.Register)
	r.Post("/api/auth/login", apiHandler.Login)
	r.Get("/api/health", apiHandler.Health)
	
	// Protected routes group
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return middleware.AuthMiddleware(next)
		})

		r.Post("/api/auth/grant-access", apiHandler.GrantAccess)
		r.Post("/api/sync", apiHandler.HandleSync)
		r.Get("/api/env-notes", apiHandler.HandleEnvNotes)
		r.Post("/api/env-notes", apiHandler.HandleEnvNotes)
		
		// Routes protected by both JWT Auth and RBAC
		r.Group(func(r chi.Router) {
			r.Use(apiHandler.RBACMiddleware)
			r.Get("/api/notes", apiHandler.HandleNotes)
			r.Post("/api/notes", apiHandler.HandleNotes)
			r.Get("/api/connections", apiHandler.HandleConnections)
			r.Post("/api/connections", apiHandler.HandleConnections)
		})
		
		// File URL routes (protected by AuthMiddleware)
		r.Post("/api/files/upload-url", apiHandler.HandleUploadURL)
		r.Get("/api/files/download-url/{id}", apiHandler.HandleDownloadURL)
	})

	// Apply Middlewares (Logging + CORS)
	var handler http.Handler = r
	handler = middleware.CORSMiddleware(handler)
	handler = middleware.LoggingMiddleware(handler)

	// HTTP Server Configuration
	serverAddr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Channel to catch server startup errors
	serverErrors := make(chan error, 1)

	// SERVER STARTUP: Run in a goroutine to avoid blocking main
	go func() {
		log.Printf("[INFO] Cozy Canvas Go API Server listening on http://localhost%s\n", serverAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Channel for system signals (process termination)
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking and waiting for events
	select {
	case err := <-serverErrors:
		log.Fatalf("[FATAL] Server failed to start: %v", err)

	case sig := <-shutdown:
		log.Printf("[INFO] Shutdown signal received (%v). Starting Graceful Shutdown...\n", sig)

		// Allocate 15 seconds to complete active client requests
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Graceful stop of HTTP server (stops accepting new, waits for existing)
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("[WARN] Server failed to shut down gracefully: %v. Forcing closure.", err)
			_ = server.Close()
		}

		// Correctly close database connection if initialized
		if db != nil {
			log.Println("[DB] Closing PostgreSQL connections...")
			if err := db.Close(); err != nil {
				log.Printf("[ERROR] Database connection closure error: %v", err)
			}
		}

		log.Println("[INFO] Cozy Canvas Backend stopped successfully.")
	}
}
