package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"cozy-canvas/backend/internal/middleware"
	"cozy-canvas/backend/internal/models"
	"cozy-canvas/backend/internal/storage"
	"cozy-canvas/backend/internal/store"
	"cozy-canvas/backend/internal/utils"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// APIHandler wraps our repository layer to expose HTTP handlers
type APIHandler struct {
	Users       store.UserRepository
	Notes       store.NoteRepository
	Connections store.ConnectionRepository
	Storage     *storage.MinIOClient
}

func NewAPIHandler(u store.UserRepository, n store.NoteRepository, c store.ConnectionRepository, s *storage.MinIOClient) *APIHandler {
	return &APIHandler{
		Users:       u,
		Notes:       n,
		Connections: c,
		Storage:     s,
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
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if ok && userID != 0 {
		return userID, nil
	}
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

// Login authenticates a user by comparing the bcrypt hash of the stored password with the password provided in the request.
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
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		a.writeJSON(w, http.StatusInternalServerError, models.AuthResponse{Status: "error", Message: "JWT secret not configured on server"})
		return
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

// Health is the readiness probe endpoint
func (a *APIHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	a.writeJSON(w, http.StatusOK, map[string]string{"status": "OK", "time": time.Now().Format(time.RFC3339)})
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

	targetUserID := userID
	if targetParam := r.URL.Query().Get("user_id"); targetParam != "" {
		var parsed int
		if _, err := fmt.Sscanf(targetParam, "%d", &parsed); err == nil {
			targetUserID = parsed
		}
	}

	switch r.Method {
	case http.MethodGet:
		notes, err := a.Notes.GetNotes(targetUserID)
		if err != nil {
			a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		// Populate HTML on-the-fly for response
		for i := range notes {
			notes[i].HTML = utils.RenderMarkdown(notes[i].Text)
		}

		a.writeJSON(w, http.StatusOK, notes)

	case http.MethodPost:
		if targetUserID != userID {
			a.writeJSON(w, http.StatusForbidden, map[string]string{"error": "Write access denied"})
			return
		}
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

		// Populate HTML on-the-fly for response
		for i := range envList {
			envList[i].HTML = utils.RenderMarkdown(envList[i].Text)
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

	targetUserID := userID
	if targetParam := r.URL.Query().Get("user_id"); targetParam != "" {
		var parsed int
		if _, err := fmt.Sscanf(targetParam, "%d", &parsed); err == nil {
			targetUserID = parsed
		}
	}

	switch r.Method {
	case http.MethodGet:
		conns, err := a.Connections.GetConnections(targetUserID)
		if err != nil {
			a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		a.writeJSON(w, http.StatusOK, conns)

	case http.MethodPost:
		if targetUserID != userID {
			a.writeJSON(w, http.StatusForbidden, map[string]string{"error": "Write access denied"})
			return
		}
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

// ==========================================================================
// RBAC & Access Grant Handlers
// ==========================================================================

func (a *APIHandler) GrantAccess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.writeJSON(w, http.StatusMethodNotAllowed, models.AuthResponse{Status: "error", Message: "Method not allowed"})
		return
	}

	viewerUserID, err := a.getUserID(r)
	if err != nil {
		a.writeJSON(w, http.StatusUnauthorized, models.AuthResponse{Status: "error", Message: "Unauthorized viewer"})
		return
	}

	var req models.GrantAccessRequest
	if err := a.readJSON(r, &req); err != nil {
		a.writeJSON(w, http.StatusBadRequest, models.AuthResponse{Status: "error", Message: "Invalid request body"})
		return
	}

	owner, err := a.Users.GetUserByEmail(req.Email)
	if err != nil {
		a.writeJSON(w, http.StatusUnauthorized, models.AuthResponse{Status: "error", Message: "Invalid owner credentials or codeword"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(owner.PasswordHash), []byte(req.Password)); err != nil {
		a.writeJSON(w, http.StatusUnauthorized, models.AuthResponse{Status: "error", Message: "Invalid owner credentials or codeword"})
		return
	}

	if req.Codeword == "" || (req.Codeword != owner.CodeWord1 && req.Codeword != owner.CodeWord2) {
		a.writeJSON(w, http.StatusUnauthorized, models.AuthResponse{Status: "error", Message: "Invalid owner credentials or codeword"})
		return
	}

	if owner.ID == viewerUserID {
		a.writeJSON(w, http.StatusBadRequest, models.AuthResponse{Status: "error", Message: "Cannot grant access to yourself"})
		return
	}

	expiresAt := time.Now().Add(time.Hour)
	if err := a.Users.CreateAccessGrant(owner.ID, viewerUserID, expiresAt); err != nil {
		a.writeJSON(w, http.StatusInternalServerError, models.AuthResponse{Status: "error", Message: "Failed to create access grant"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"owner_user_id":  owner.ID,
		"viewer_user_id": viewerUserID,
		"exp":            expiresAt.Unix(),
	})

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		a.writeJSON(w, http.StatusInternalServerError, models.AuthResponse{Status: "error", Message: "JWT secret not configured on server"})
		return
	}

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		a.writeJSON(w, http.StatusInternalServerError, models.AuthResponse{Status: "error", Message: "Failed to sign grant token"})
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "success",
		"message":       "Access grant created successfully",
		"token":         tokenString,
		"expires_at":    expiresAt.Format(time.RFC3339),
		"owner_user_id": owner.ID,
	})
}

func (a *APIHandler) RBACMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := a.getUserID(r)
		if err != nil {
			a.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
			return
		}

		targetParam := r.URL.Query().Get("user_id")
		if targetParam != "" {
			var targetUserID int
			if _, err := fmt.Sscanf(targetParam, "%d", &targetUserID); err == nil && targetUserID != userID {
				if r.Method != http.MethodGet {
					a.writeJSON(w, http.StatusForbidden, map[string]string{"error": "Forbidden: Write access denied"})
					return
				}

				allowed, err := a.Users.VerifyAccessGrant(targetUserID, userID)
				if err != nil || !allowed {
					a.writeJSON(w, http.StatusForbidden, map[string]string{"error": "Forbidden: No valid access grant"})
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// ==========================================================================
// File Upload & Download presigned URL Handlers
// ==========================================================================

func (a *APIHandler) HandleUploadURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		Filename string `json:"filename"`
	}
	if err := a.readJSON(r, &req); err != nil || req.Filename == "" {
		a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Filename is required"})
		return
	}

	// Clean filename using path.Base and filepath.ToSlash to prevent Path Traversal
	cleanFilename := path.Base(filepath.ToSlash(req.Filename))
	cleanFilename = strings.ReplaceAll(cleanFilename, "..", "")
	if cleanFilename == "." || cleanFilename == "/" || cleanFilename == "" {
		a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid filename"})
		return
	}

	// Generate a unique object name in bucket: format "unixnano-filename" to prevent duplicates
	objectName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), cleanFilename)

	if a.Storage == nil {
		a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Storage service not initialized"})
		return
	}

	// Generate put url valid for 15 minutes
	uploadURL, err := a.Storage.GeneratePresignedPutURL(r.Context(), objectName, 15*time.Minute)
	if err != nil {
		a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]string{
		"uploadUrl": uploadURL,
		"filename":  objectName,
	})
}

func (a *APIHandler) HandleDownloadURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	// Safely retrieve the file ID path parameter using Go Chi
	objectName := chi.URLParam(r, "id")
	if objectName == "" {
		a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "File ID is required"})
		return
	}

	if a.Storage == nil {
		a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Storage service not initialized"})
		return
	}

	// Generate get url valid for 1 hour
	downloadURL, err := a.Storage.GeneratePresignedGetURL(r.Context(), objectName, time.Hour)
	if err != nil {
		a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]string{
		"downloadUrl": downloadURL,
	})
}

// HandleSync implements the /api/sync endpoint for atomic save of the notes & connections graph
func (a *APIHandler) HandleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	userID, err := a.getUserID(r)
	if err != nil {
		a.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "User not found"})
		return
	}

	var req models.SyncRequest
	if err := a.readJSON(r, &req); err != nil {
		a.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	err = a.Notes.SyncGraph(userID, req.Notes, req.Connections)
	if err != nil {
		a.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	a.writeJSON(w, http.StatusOK, map[string]string{"status": "synchronized"})
}
