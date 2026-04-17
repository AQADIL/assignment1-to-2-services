package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"payment-service/internal/domain"

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
CREATE TABLE IF NOT EXISTS payments (
  id TEXT PRIMARY KEY,
  order_id TEXT NOT NULL UNIQUE,
  transaction_id TEXT NOT NULL,
  amount INTEGER NOT NULL,
  status TEXT NOT NULL
);
`
	_, err := r.db.Exec(q)
	return err
}

func (r *SQLiteRepository) Create(ctx context.Context, p domain.Payment) error {
	q := `INSERT INTO payments (id, order_id, transaction_id, amount, status) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.OrderID, p.TransactionID, p.Amount, p.Status)
	return err
}

func (r *SQLiteRepository) GetByOrderID(ctx context.Context, orderID string) (domain.Payment, error) {
	q := `SELECT id, order_id, transaction_id, amount, status FROM payments WHERE order_id = ?`
	row := r.db.QueryRowContext(ctx, q, orderID)

	var p domain.Payment
	if err := row.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Payment{}, domain.ErrPaymentNotFound
		}
		return domain.Payment{}, err
	}
	return p, nil
}

func (r *SQLiteRepository) FindByAmountRange(ctx context.Context, minAmount, maxAmount int64) ([]domain.Payment, error) {
	q := `SELECT id, order_id, transaction_id, amount, status FROM payments WHERE (? = 0 OR amount >= ?) AND (? = 0 OR amount <= ?)`
	rows, err := r.db.QueryContext(ctx, q, minAmount, minAmount, maxAmount, maxAmount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return payments, nil
}
