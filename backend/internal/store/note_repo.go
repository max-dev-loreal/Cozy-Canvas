package store

import (
	"cozy-canvas/backend/internal/models"
	"database/sql"
)

type pgNoteRepository struct {
	db *sql.DB
}

func NewPGNoteRepository(db *sql.DB) NoteRepository {
	return &pgNoteRepository{db: db}
}

func (r *pgNoteRepository) GetNotes(userID int) ([]models.Note, error) {
	rows, err := r.db.Query("SELECT id, text, x, y, is_env FROM notes WHERE user_id = $1 AND is_env = false", userID)
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

func (r *pgNoteRepository) SaveNotes(userID int, notes []models.Note) error {
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
		_, err = tx.Exec("INSERT INTO notes (id, user_id, text, x, y, is_env) VALUES ($1, $2, $3, $4, $5, false)",
			n.ID, userID, n.Text, n.X, n.Y)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *pgNoteRepository) GetEnvNotes() ([]models.Note, error) {
	rows, err := r.db.Query("SELECT id, text, x, y, is_env FROM notes WHERE is_env = true AND user_id IS NULL")
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

func (r *pgNoteRepository) SaveEnvNotes(notes []models.Note) error {
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
		_, err = tx.Exec("INSERT INTO notes (id, user_id, text, x, y, is_env) VALUES ($1, NULL, $2, $3, $4, true)",
			n.ID, n.Text, n.X, n.Y)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
