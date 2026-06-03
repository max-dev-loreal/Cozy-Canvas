package store

import (
	"cozy-canvas/backend/internal/models"
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type pgUserRepository struct {
	db *sql.DB
}

func NewPGUserRepository(db *sql.DB) UserRepository {
	return &pgUserRepository{db: db}
}

// RegisterUser hashes the user's password using bcrypt before inserting it into the database to ensure plaintext passwords are never stored.
func (r *pgUserRepository) RegisterUser(username, email, password, word1, word2 string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = r.db.Exec("INSERT INTO users (username, email, password_hash, code_word1, code_word2) VALUES ($1, $2, $3, $4, $5)",
		username, email, string(hashedPassword), word1, word2)
	return err
}

func (r *pgUserRepository) GetUserByEmail(email string) (models.User, error) {
	var u models.User
	err := r.db.QueryRow("SELECT id, username, email, password_hash, code_word1, code_word2 FROM users WHERE email = $1", email).
		Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CodeWord1, &u.CodeWord2)
	if err != nil {
		return models.User{}, err
	}
	return u, nil
}

func (r *pgUserRepository) GetUserIDByUsername(username string) (int, error) {
	var id int
	err := r.db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&id)
	return id, err
}

func (r *pgUserRepository) CreateAccessGrant(ownerUserID, viewerUserID int, expiresAt time.Time) error {
	// Clean up any existing grants between these two users first
	_, _ = r.db.Exec("DELETE FROM access_grants WHERE owner_user_id = $1 AND viewer_user_id = $2", ownerUserID, viewerUserID)

	_, err := r.db.Exec("INSERT INTO access_grants (owner_user_id, viewer_user_id, code_word_verified, expires_at) VALUES ($1, $2, true, $3)",
		ownerUserID, viewerUserID, expiresAt)
	return err
}

func (r *pgUserRepository) VerifyAccessGrant(ownerUserID, viewerUserID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM access_grants 
			WHERE owner_user_id = $1 AND viewer_user_id = $2 AND expires_at > NOW()
		)`, ownerUserID, viewerUserID).Scan(&exists)
	return exists, err
}
