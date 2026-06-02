package store

import (
	"cozy-canvas/backend/internal/models"
	"database/sql"
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
