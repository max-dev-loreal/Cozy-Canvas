package store

import (
	"cozy-canvas/backend/internal/models"
	"time"
)

// UserRepository handles user data persistence
type UserRepository interface {
	RegisterUser(username, email, password, word1, word2 string) error
	GetUserByEmail(email string) (models.User, error)
	GetUserIDByUsername(username string) (int, error)
	CreateAccessGrant(ownerUserID, viewerUserID int, expiresAt time.Time) error
	VerifyAccessGrant(ownerUserID, viewerUserID int) (bool, error)
}

// NoteRepository handles canvas notes persistence
type NoteRepository interface {
	GetNotes(userID int) ([]models.Note, error)
	SaveNotes(userID int, notes []models.Note) error
	GetEnvNotes() ([]models.Note, error)
	SaveEnvNotes(notes []models.Note) error
	SyncGraph(userID int, notes []models.Note, conns []models.Connection) error
}

// ConnectionRepository handles node connections persistence
type ConnectionRepository interface {
	GetConnections(userID int) ([]models.Connection, error)
	SaveConnections(userID int, conns []models.Connection) error
}
