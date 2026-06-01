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
	var store handlers.Store
	var db *sql.DB
	var err error

	// FIXED: Removed nil check that caused compilation error
	if dbURL != "" {
		log.Printf("[DB] Connecting to PostgreSQL at %s...\n", dbURL)
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Printf("[ERROR] Database connection failed: %v. Falling back to MemoryStore.\n", err)
			store = handlers.NewMemoryStore()
		} else {
			// Verify db connection
			err = db.Ping()
			if err != nil {
				log.Printf("[ERROR] Database ping failed: %v. Falling back to MemoryStore.\n", err)
				store = handlers.NewMemoryStore()
			} else {
				log.Println("[DB] Successfully connected to PostgreSQL! Active production mode.")
				store = handlers.NewDBStore(db)
			}
		}
	} else {
		log.Println("[INFO] DATABASE_URL env variable not provided. Starting in MemoryStore (Fallback Mode).")
		store = handlers.NewMemoryStore()
	}

	// Initialize API handlers
	apiHandler := handlers.NewAPIHandler(store)

	// Create custom ServeMux for standard routing
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/api/auth/register", apiHandler.Register)
	mux.HandleFunc("/api/auth/login", apiHandler.Login)
	
	// Protected routes
	mux.Handle("/api/notes", middleware.AuthMiddleware(http.HandlerFunc(apiHandler.HandleNotes)))
	mux.Handle("/api/env-notes", middleware.AuthMiddleware(http.HandlerFunc(apiHandler.HandleEnvNotes)))
	mux.Handle("/api/connections", middleware.AuthMiddleware(http.HandlerFunc(apiHandler.HandleConnections)))

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
