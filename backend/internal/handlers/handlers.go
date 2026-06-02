package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"cozy-canvas/backend/internal/middleware"
	"cozy-canvas/backend/internal/models"
	"cozy-canvas/backend/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// APIHandler wraps our repository layer to expose HTTP handlers
type APIHandler struct {
	Users       store.UserRepository
	Notes       store.NoteRepository
	Connections store.ConnectionRepository
}

func NewAPIHandler(u store.UserRepository, n store.NoteRepository, c store.ConnectionRepository) *APIHandler {
	return &APIHandler{
		Users:       u,
		Notes:       n,
		Connections: c,
	}
}

// ==========================================================================
// Helper functions
// ==========================================================================

func (a *APIHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (a *APIHandler) readJSON(r *http.Request, dst interface{}) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

func (a *APIHandler) getUsername(r *http.Request) string {
	username, ok := r.Context().Value(middleware.UserContextKey).(string)
	if !ok || username == "" {
		return "guest_devops"
	}
	return username
}

func (a *APIHandler) getUserID(r *http.Request) (int, error) {
	username := a.getUsername(r)
	return a.Users.GetUserIDByUsername(username)
}

// ==========================================================================
// Auth Handlers
// ==========================================================================

func (a *APIHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{Status: "error", Message: "Method not allowed"})
		return
	}

	var req models.RegisterRequest
	if err := a.readJSON(r, &req); err != nil {
		a.writeJSON(w, http.StatusBadRequest, models.AuthResponse{Status: "error", Message: "Invalid request body"})
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		a.writeJSON(w, http.StatusBadRequest, models.AuthResponse{Status: "error", Message: "Username, email, and password are required"})
		return
	}

	var word1, word2 string
	if len(req.CodeWords) > 0 {
		word1 = req.CodeWords[0]
	}
	if len(req.CodeWords) > 1 {
		word2 = req.CodeWords[1]
	}

	err := a.Users.RegisterUser(req.Username, req.Email, req.Password, word1, word2)
	if err != nil {
		a.writeJSON(w, http.StatusConflict, models.AuthResponse{Status: "error", Message: err.Error()})
		return
	}

	a.writeJSON(w, http.StatusOK, models.AuthResponse{Status: "success", Username: req.Username})
}

func (a *APIHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{Status: "error", Message: "Method not allowed"})
		return
	}

	var req models.LoginRequest
	if err := a.readJSON(r, &req); err != nil {
		a.writeJSON(w, http.StatusBadRequest, models.AuthResponse{Status: "error", Message: "Invalid request body"})
		return
	}

	user, err := a.Users.GetUserByEmail(req.Email)
	if err != nil {
		a.writeJSON(w, http.StatusUnauthorized, models.AuthResponse{Status: "error", Message: "Invalid email or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		a.writeJSON(w, http.StatusUnauthorized, models.AuthResponse{Status: "error", Message: "Invalid email or password"})
		return
	}

	// Generate JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-secret-change-me"
	}

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		a.writeJSON(w, http.StatusInternalServerError, models.AuthResponse{Status: "error", Message: "Could not generate token"})
		return
	}

	a.writeJSON(w, http.StatusOK, models.AuthResponse{
		Status:   "success",
		Username: user.Username,
		Token:    tokenString,
	})
}

// ==========================================================================
// Notes Handlers
// ==========================================================================

func (a *APIHandler) HandleNotes(w http.ResponseWriter, r *http.Request) {
	userID, err := a.getUserID(r)
	if err != nil {
		a.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "User not found"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		notes, err := a.Notes.GetNotes(userID)
		if err != nil {
			a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, notes)

	case http.MethodPost:
		var notesList []models.Note
		if err := a.readJSON(r, &notesList); err != nil {
			a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON notes payload"})
			return
		}
		
		err := a.Notes.SaveNotes(userID, notesList)
		if err != nil {
			a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})

	default:
		a.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

// ==========================================================================
// Global Env Handlers
// ==========================================================================

func (a *APIHandler) HandleEnvNotes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		envList, err := a.Notes.GetEnvNotes()
		if err != nil {
			a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, envList)

	case http.MethodPost:
		var envList []models.Note
		if err := a.readJSON(r, &envList); err != nil {
			a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON env-notes payload"})
			return
		}

		err := a.Notes.SaveEnvNotes(envList)
		if err != nil {
			a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})

	default:
		a.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

// ==========================================================================
// Connections Handlers
// ==========================================================================

func (a *APIHandler) HandleConnections(w http.ResponseWriter, r *http.Request) {
	userID, err := a.getUserID(r)
	if err != nil {
		a.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "User not found"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		conns, err := a.Connections.GetConnections(userID)
		if err != nil {
			a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, conns)

	case http.MethodPost:
		var connsList []models.Connection
		if err := a.readJSON(r, &connsList); err != nil {
			a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON connections payload"})
			return
		}

		err := a.Connections.SaveConnections(userID, connsList)
		if err != nil {
			a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})

	default:
		a.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}
