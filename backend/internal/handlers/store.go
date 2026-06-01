package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"cozy-canvas/backend/internal/models"

	"golang.org/x/crypto/bcrypt"
)

// Store defines the storage engine interface for Cozy Canvas
type Store interface {
	GetNotes(username string) ([]models.Note, error)
	SaveNotes(username string, notes []models.Note) error
	GetEnvNotes() ([]models.Note, error)
	SaveEnvNotes(notes []models.Note) error
	GetConnections(username string) ([]models.Connection, error)
	SaveConnections(username string, conns []models.Connection) error
	RegisterUser(username, email, password, word1, word2 string) error
	GetUserByEmail(email string) (models.User, error)
}

// ==========================================================================
// MemoryStore - Thread-safe in-memory fallback store
// ==========================================================================

type MemoryStore struct {
	sync.RWMutex
	users       map[string]models.User       // key: email
	notes       map[string][]models.Note     // key: username
	envNotes    []models.Note
	connections map[string][]models.Connection // key: username
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users: make(map[string]models.User),
		notes: make(map[string][]models.Note),
		envNotes: []models.Note{
			{ID: "env-USER", Text: "⚙️ USER\n\nguest_devops", X: 400, Y: 100, IsEnv: true},
			{ID: "env-OS", Text: "⚙️ OS\n\nNixOS", X: 600, Y: 300, IsEnv: true},
			{ID: "env-ENV", Text: "⚙️ ENV\n\nproduction", X: 200, Y: 300, IsEnv: true},
		},
		connections: make(map[string][]models.Connection),
	}
}

func (m *MemoryStore) GetNotes(username string) ([]models.Note, error) {
	m.RLock()
	defer m.RUnlock()
	list, exists := m.notes[username]
	if !exists {
		return []models.Note{}, nil
	}
	return list, nil
}

func (m *MemoryStore) SaveNotes(username string, notes []models.Note) error {
	m.Lock()
	defer m.Unlock()
	m.notes[username] = notes
	return nil
}

func (m *MemoryStore) GetEnvNotes() ([]models.Note, error) {
	m.RLock()
	defer m.RUnlock()
	return m.envNotes, nil
}

func (m *MemoryStore) SaveEnvNotes(notes []models.Note) error {
	m.Lock()
	defer m.Unlock()
	m.envNotes = notes
	return nil
}

func (m *MemoryStore) GetConnections(username string) ([]models.Connection, error) {
	m.RLock()
	defer m.RUnlock()
	list, exists := m.connections[username]
	if !exists {
		return []models.Connection{}, nil
	}
	return list, nil
}

func (m *MemoryStore) SaveConnections(username string, conns []models.Connection) error {
	m.Lock()
	defer m.Unlock()
	m.connections[username] = conns
	return nil
}

func (m *MemoryStore) RegisterUser(username, email, password, word1, word2 string) error {
	m.Lock()
	defer m.Unlock()

	for _, u := range m.users {
		if u.Username == username {
			return errors.New("username already exists")
		}
		if u.Email == email {
			return errors.New("email already registered")
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	newUser := models.User{
		ID:           len(m.users) + 1,
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		CodeWord1:    word1,
		CodeWord2:    word2,
		CreatedAt:    time.Now(),
	}
	m.users[email] = newUser
	return nil
}

func (m *MemoryStore) GetUserByEmail(email string) (models.User, error) {
	m.RLock()
	defer m.RUnlock()
	u, exists := m.users[email]
	if !exists {
		return models.User{}, errors.New("user not found")
	}
	return u, nil
}

// ==========================================================================
// DBStore - PostgreSQL storage backend
// ==========================================================================

type DBStore struct {
	db *sql.DB
}

func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

func (d *DBStore) GetNotes(username string) ([]models.Note, error) {
	rows, err := d.db.Query("SELECT id, text, x, y, is_env FROM notes WHERE user_id = $1 AND is_env = false", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Note
	for rows.Next() {
		var n models.Note
		n.UserID = username
		if err := rows.Scan(&n.ID, &n.Text, &n.X, &n.Y, &n.IsEnv); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	if list == nil {
		list = []models.Note{}
	}
	return list, nil
}

func (d *DBStore) SaveNotes(username string, notes []models.Note) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clean existing notes
	_, err = tx.Exec("DELETE FROM notes WHERE user_id = $1 AND is_env = false", username)
	if err != nil {
		return err
	}

	// Insert notes
	for _, n := range notes {
		_, err = tx.Exec("INSERT INTO notes (id, user_id, text, x, y, is_env) VALUES ($1, $2, $3, $4, $5, false)",
			n.ID, username, n.Text, n.X, n.Y)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DBStore) GetEnvNotes() ([]models.Note, error) {
	rows, err := d.db.Query("SELECT id, text, x, y, is_env FROM notes WHERE is_env = true")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Note
	for rows.Next() {
		var n models.Note
		if err := rows.Scan(&n.ID, &n.Text, &n.X, &n.Y, &n.IsEnv); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	if list == nil {
		list = []models.Note{}
	}
	return list, nil
}

func (d *DBStore) SaveEnvNotes(notes []models.Note) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clean existing env notes
	_, err = tx.Exec("DELETE FROM notes WHERE is_env = true")
	if err != nil {
		return err
	}

	// Insert env notes
	for _, n := range notes {
		_, err = tx.Exec("INSERT INTO notes (id, user_id, text, x, y, is_env) VALUES ($1, 'global', $2, $3, $4, true)",
			n.ID, n.Text, n.X, n.Y)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DBStore) GetConnections(username string) ([]models.Connection, error) {
	rows, err := d.db.Query("SELECT id, source_note_id, target_note_id FROM connections WHERE user_id = $1", username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Connection
	for rows.Next() {
		var c models.Connection
		c.UserID = username
		if err := rows.Scan(&c.ID, &c.Source, &c.Target); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	if list == nil {
		list = []models.Connection{}
	}
	return list, nil
}

func (d *DBStore) SaveConnections(username string, conns []models.Connection) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clean existing connections
	_, err = tx.Exec("DELETE FROM connections WHERE user_id = $1", username)
	if err != nil {
		return err
	}

	// Insert connections
	for _, c := range conns {
		connID := fmt.Sprintf("%s-%s", c.Source, c.Target)
		_, err = tx.Exec("INSERT INTO connections (id, user_id, source_note_id, target_note_id) VALUES ($1, $2, $3, $4)",
			connID, username, c.Source, c.Target)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (d *DBStore) RegisterUser(username, email, password, word1, word2 string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = d.db.Exec("INSERT INTO users (username, email, password_hash, code_word1, code_word2) VALUES ($1, $2, $3, $4, $5)",
		username, email, string(hashedPassword), word1, word2)
	return err
}

func (d *DBStore) GetUserByEmail(email string) (models.User, error) {
	var u models.User
	err := d.db.QueryRow("SELECT id, username, email, password_hash, code_word1, code_word2 FROM users WHERE email = $1", email).
		Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.CodeWord1, &u.CodeWord2)
	if err != nil {
		return models.User{}, err
	}
	return u, nil
}
