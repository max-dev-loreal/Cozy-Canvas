package store

import (
	"cozy-canvas/backend/internal/models"
	"database/sql"
	"fmt"
)

type pgNoteRepository struct {
	db *sql.DB
}

func NewPGNoteRepository(db *sql.DB) NoteRepository {
	return &pgNoteRepository{db: db}
}

func (r *pgNoteRepository) GetNotes(userID int) ([]models.Note, error) {
	rows, err := r.db.Query("SELECT id, text, context, x, y, is_env FROM notes WHERE user_id = $1 AND is_env = false", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Note
	for rows.Next() {
		var n models.Note
		if err := rows.Scan(&n.ID, &n.Text, &n.Context, &n.X, &n.Y, &n.IsEnv); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	if list == nil {
		list = []models.Note{}
	}
	return list, nil
}

func (r *pgNoteRepository) SaveNotes(userID int, notes []models.Note) error {
	// Start SQL transaction to ensure atomic deletion and bulk insertion of notes
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM notes WHERE user_id = $1 AND is_env = false", userID)
	if err != nil {
		return err
	}

	for _, n := range notes {
		_, err = tx.Exec("INSERT INTO notes (id, user_id, text, context, x, y, is_env) VALUES ($1, $2, $3, $4, $5, $6, false)",
			n.ID, userID, n.Text, n.Context, n.X, n.Y)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *pgNoteRepository) GetEnvNotes() ([]models.Note, error) {
	rows, err := r.db.Query("SELECT id, text, context, x, y, is_env FROM notes WHERE is_env = true AND user_id IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Note
	for rows.Next() {
		var n models.Note
		if err := rows.Scan(&n.ID, &n.Text, &n.Context, &n.X, &n.Y, &n.IsEnv); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	if list == nil {
		list = []models.Note{}
	}
	return list, nil
}

func (r *pgNoteRepository) SaveEnvNotes(notes []models.Note) error {
	// Start SQL transaction to ensure atomic deletion and bulk insertion of env notes
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM notes WHERE is_env = true AND user_id IS NULL")
	if err != nil {
		return err
	}

	for _, n := range notes {
		_, err = tx.Exec("INSERT INTO notes (id, user_id, text, context, x, y, is_env) VALUES ($1, NULL, $2, $3, $4, $5, true)",
			n.ID, n.Text, n.Context, n.X, n.Y)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *pgNoteRepository) SyncGraph(userID int, notes []models.Note, conns []models.Connection) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Delete all connections of the user first to satisfy foreign keys before deleting notes
	_, err = tx.Exec("DELETE FROM connections WHERE user_id = $1", userID)
	if err != nil {
		return err
	}

	// 2. Delete all notes of the user
	_, err = tx.Exec("DELETE FROM notes WHERE user_id = $1 AND is_env = false", userID)
	if err != nil {
		return err
	}

	// 3. Insert new notes (ignoring any pre-existing user_id, binding strictly to the authenticated JWT userID)
	for _, n := range notes {
		_, err = tx.Exec("INSERT INTO notes (id, user_id, text, context, x, y, is_env) VALUES ($1, $2, $3, $4, $5, $6, false)",
			n.ID, userID, n.Text, n.Context, n.X, n.Y)
		if err != nil {
			return err
		}
	}

	// 4. Insert new connections
	for _, c := range conns {
		connID := fmt.Sprintf("%s-%s", c.Source, c.Target)
		_, err = tx.Exec("INSERT INTO connections (id, user_id, source_note_id, target_note_id) VALUES ($1, $2, $3, $4)",
			connID, userID, c.Source, c.Target)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
