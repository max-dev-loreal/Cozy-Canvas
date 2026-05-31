package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

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

	if dbURL != nil && dbURL != "" {
		log.Printf("[DB] Connecting to PostgreSQL at %s...\n", dbURL)
		db, err := sql.Open("postgres", dbURL)
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
	mux.HandleFunc("/api/notes", apiHandler.HandleNotes)
	mux.HandleFunc("/api/env-notes", apiHandler.HandleEnvNotes)
	mux.HandleFunc("/api/connections", apiHandler.HandleConnections)

	// Apply Middlewares (Logging + CORS)
	var handler http.Handler = mux
	handler = middleware.CORSMiddleware(handler)
	handler = middleware.LoggingMiddleware(handler)

	// Start Server
	serverAddr := fmt.Sprintf(":%s", port)
	log.Printf("[INFO] Cozy Canvas Go API Server listening on http://localhost%s\n", serverAddr)
	log.Println("[INFO] Press Ctrl+C to terminate.")

	if err := http.ListenAndServe(serverAddr, handler); err != nil {
		log.Fatalf("[FATAL] Server failed to start: %v", err)
	}
}
