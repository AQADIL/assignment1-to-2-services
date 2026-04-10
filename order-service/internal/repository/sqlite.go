package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"order-service/internal/domain"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type subscriber struct {
	ch         chan domain.Order
	customerID string
}

type SQLiteRepository struct {
	db          *sql.DB
	mu          sync.RWMutex
	subscribers map[string]*subscriber
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

	repo := &SQLiteRepository{db: db, subscribers: make(map[string]*subscriber)}
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
CREATE TABLE IF NOT EXISTS orders (
  id TEXT PRIMARY KEY,
  customer_id TEXT NOT NULL,
  item_name TEXT NOT NULL,
  amount INTEGER NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
  key TEXT PRIMARY KEY,
  status_code INTEGER NOT NULL,
  response_body BLOB NOT NULL,
  created_at TEXT NOT NULL
);
`
	_, err := r.db.Exec(q)
	return err
}

func (r *SQLiteRepository) Create(ctx context.Context, order domain.Order) error {
	q := `INSERT INTO orders (id, customer_id, item_name, amount, status, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, q, order.ID, order.CustomerID, order.ItemName, order.Amount, order.Status, order.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (domain.Order, error) {
	q := `SELECT id, customer_id, item_name, amount, status, created_at FROM orders WHERE id = ?`
	row := r.db.QueryRowContext(ctx, q, id)

	var o domain.Order
	var createdAt string
	if err := row.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Order{}, domain.ErrOrderNotFound
		}
		return domain.Order{}, err
	}
	t, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return domain.Order{}, err
	}
	o.CreatedAt = t
	return o, nil
}

func (r *SQLiteRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	q := `UPDATE orders SET status = ? WHERE id = ?`
	res, err := r.db.ExecContext(ctx, q, status, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrOrderNotFound
	}

	if order, err := r.GetByID(ctx, id); err == nil {
		r.notify(order)
	}
	return nil
}

func (r *SQLiteRepository) Subscribe(customerID string) (<-chan domain.Order, string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	subID := uuid.NewString()
	ch := make(chan domain.Order, 64)
	r.subscribers[subID] = &subscriber{ch: ch, customerID: customerID}
	return ch, subID
}

func (r *SQLiteRepository) Unsubscribe(subID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if sub, ok := r.subscribers[subID]; ok {
		close(sub.ch)
		delete(r.subscribers, subID)
	}
}

func (r *SQLiteRepository) notify(order domain.Order) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, sub := range r.subscribers {
		if sub.customerID == "" || sub.customerID == order.CustomerID {
			select {
			case sub.ch <- order:
			default:
			}
		}
	}
}

func (r *SQLiteRepository) ListByCustomerID(ctx context.Context, customerID string) ([]domain.Order, error) {
	q := `SELECT id, customer_id, item_name, amount, status, created_at FROM orders WHERE customer_id = ? ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		var createdAt string
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &createdAt); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		o.CreatedAt = t
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *SQLiteRepository) Get(ctx context.Context, key string) (statusCode int, body []byte, found bool, err error) {
	q := `SELECT status_code, response_body FROM idempotency_keys WHERE key = ?`
	row := r.db.QueryRowContext(ctx, q, key)
	if err := row.Scan(&statusCode, &body); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil, false, nil
		}
		return 0, nil, false, err
	}
	return statusCode, body, true, nil
}

func (r *SQLiteRepository) Save(ctx context.Context, key string, statusCode int, body []byte) error {
	q := `INSERT OR IGNORE INTO idempotency_keys (key, status_code, response_body, created_at) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, q, key, statusCode, body, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
