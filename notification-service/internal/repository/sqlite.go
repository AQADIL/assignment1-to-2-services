package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", dbPath))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	repo := &SQLiteRepository{db: db}
	if err := repo.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repo, nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

func (r *SQLiteRepository) migrate() error {
	q := `
CREATE TABLE IF NOT EXISTS processed_events (
  event_id TEXT PRIMARY KEY,
  created_at DATETIME NOT NULL
);
`
	_, err := r.db.Exec(q)
	return err
}

func (r *SQLiteRepository) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	q := `SELECT 1 FROM processed_events WHERE event_id = ? LIMIT 1`
	row := r.db.QueryRowContext(ctx, q, eventID)
	var one int
	if err := row.Scan(&one); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *SQLiteRepository) MarkProcessed(ctx context.Context, eventID string) error {
	q := `INSERT INTO processed_events (event_id, created_at) VALUES (?, ?)`
	_, err := r.db.ExecContext(ctx, q, eventID, time.Now().UTC())
	return err
}
