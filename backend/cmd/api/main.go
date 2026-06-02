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

	var minioClient *storage.MinIOClient
	if minioEndpoint != "" && minioAccessKey != "" && minioSecretKey != "" && minioBucket != "" {
		log.Printf("[MinIO] Connecting to MinIO endpoint %s (bucket: %s)...\n", minioEndpoint, minioBucket)
		minioClient, err = storage.NewMinIOClient(minioEndpoint, minioAccessKey, minioSecretKey, minioBucket)
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

	// Create custom ServeMux for standard routing
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/api/auth/register", apiHandler.Register)
	mux.HandleFunc("/api/auth/login", apiHandler.Login)
	mux.HandleFunc("/api/health", apiHandler.Health)
	
	// Protected routes
	mux.Handle("/api/auth/grant-access", middleware.AuthMiddleware(http.HandlerFunc(apiHandler.GrantAccess)))
	mux.Handle("/api/notes", middleware.AuthMiddleware(apiHandler.RBACMiddleware(http.HandlerFunc(apiHandler.HandleNotes))))
	mux.Handle("/api/env-notes", middleware.AuthMiddleware(http.HandlerFunc(apiHandler.HandleEnvNotes)))
	mux.Handle("/api/connections", middleware.AuthMiddleware(apiHandler.RBACMiddleware(http.HandlerFunc(apiHandler.HandleConnections))))
	
	// File URL routes (protected by AuthMiddleware)
	mux.Handle("/api/files/upload-url", middleware.AuthMiddleware(http.HandlerFunc(apiHandler.HandleUploadURL)))
	mux.Handle("/api/files/download-url/", middleware.AuthMiddleware(http.HandlerFunc(apiHandler.HandleDownloadURL)))

	// Apply Middlewares (Logging + CORS)
	var handler http.Handler = mux
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
