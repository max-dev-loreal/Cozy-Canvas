package models

import "time"

// User represents an engineer profile in Cozy Cluster
type User struct {
	ID           int       `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CodeWord1    string    `json:"-" db:"code_word1"`
	CodeWord2    string    `json:"-" db:"code_word2"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Note represents an interactive node on the canvas
type Note struct {
	ID        string    `json:"id" db:"id"`
	UserID    *int      `json:"user_id,omitempty" db:"user_id"` // Pointer because it can be NULL for global envs
	Text      string    `json:"text" db:"text"`
	X         float64   `json:"x" db:"x"`
	Y         float64   `json:"y" db:"y"`
	IsEnv     bool      `json:"isEnv" db:"is_env"`
	CreatedAt time.Time `json:"created_at,omitempty" db:"created_at"`
}

// Connection represents a physics link between two notes
type Connection struct {
	ID        string    `json:"id" db:"id"` // Format: source-target
	UserID    int       `json:"user_id,omitempty" db:"user_id"`
	Source    string    `json:"source" db:"source_note_id"`
	Target    string    `json:"target" db:"target_note_id"`
	CreatedAt time.Time `json:"created_at,omitempty" db:"created_at"`
}

// RegisterRequest represents the JSON request payload for registration
type RegisterRequest struct {
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Password  string   `json:"password"`
	CodeWords []string `json:"codewords"`
}

// LoginRequest represents the JSON request payload for logging in
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents the API response for authentication
type AuthResponse struct {
	Status   string `json:"status"`
	Username string `json:"username,omitempty"`
	Message  string `json:"message,omitempty"`
	Token    string `json:"token,omitempty"`
}
