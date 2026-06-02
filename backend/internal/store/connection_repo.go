package store

import (
	"cozy-canvas/backend/internal/models"
	"database/sql"
	"fmt"
)

type pgConnectionRepository struct {
	db *sql.DB
}

func NewPGConnectionRepository(db *sql.DB) ConnectionRepository {
	return &pgConnectionRepository{db: db}
}

func (r *pgConnectionRepository) GetConnections(userID int) ([]models.Connection, error) {
	rows, err := r.db.Query("SELECT id, source_note_id, target_note_id FROM connections WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Connection
	for rows.Next() {
		var c models.Connection
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

func (r *pgConnectionRepository) SaveConnections(userID int, conns []models.Connection) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM connections WHERE user_id = $1", userID)
	if err != nil {
		return err
	}

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
