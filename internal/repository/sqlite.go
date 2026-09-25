package repository

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/junnotantra/backend-test/internal/model"
)

type SQLite struct {
	db     *sql.DB
	locker sync.Mutex
}

func NewSQLite(db *sql.DB) *SQLite { return &SQLite{db: db} }

func Open(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}
	return NewSQLite(db), nil
}

func (r *SQLite) DB() *sql.DB  { return r.db }
func (r *SQLite) Close() error { return r.db.Close() }

func (r *SQLite) CreateItem(ctx context.Context, sku, name string, quantity int64) (model.Item, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Item{}, err
	}

	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `INSERT INTO items (sku, name) VALUES (?, ?)`, sku, name)
	if err != nil {
		return model.Item{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return model.Item{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO inventory (item_id, quantity) VALUES (?, ?)`, id, quantity); err != nil {
		return model.Item{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Item{}, err
	}

	return r.GetItem(ctx, id)
}

func (r *SQLite) ListItems(ctx context.Context) ([]model.Item, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id, i.sku, i.name, inv.quantity, i.created_at
		FROM items i JOIN inventory inv ON inv.item_id = i.id
		ORDER BY i.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.Item{}
	for rows.Next() {
		var current model.Item
		if err := rows.Scan(&current.ID, &current.SKU, &current.Name, &current.Quantity, &current.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, current)
	}
	return items, rows.Err()
}

func (r *SQLite) GetItem(ctx context.Context, id int64) (model.Item, error) {
	var current model.Item
	err := r.db.QueryRowContext(ctx, `
		SELECT i.id, i.sku, i.name, inv.quantity, i.created_at
		FROM items i JOIN inventory inv ON inv.item_id = i.id WHERE i.id = ?`, id).
		Scan(&current.ID, &current.SKU, &current.Name, &current.Quantity, &current.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Item{}, model.ErrNotFound
	}
	return current, err
}

func (r *SQLite) AdjustStock(ctx context.Context, id, delta int64) (model.Item, error) {
	var newQty int64
	r.locker.Lock()
	defer r.locker.Unlock()
	err := r.db.QueryRowContext(ctx, `UPDATE inventory SET quantity = quantity + ? WHERE item_id = ? AND quantity + ? >= 0 RETURNING quantity`, delta, id, delta).Scan(&newQty)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Item{}, model.ErrNegativeStock
	}

	if err != nil {
		return model.Item{}, err
	}

	updated, err := r.GetItem(ctx, id)
	if err != nil {
		return model.Item{}, err
	}

	updated.Quantity = newQty

	return updated, nil
}

func (r *SQLite) GetItemBySKU(ctx context.Context, sku string) (model.Item, error) {
	var current model.Item
	err := r.db.QueryRowContext(ctx, `
		SELECT i.id, i.sku, i.name, inv.quantity, i.created_at
		FROM items i JOIN inventory inv ON inv.item_id = i.id WHERE i.sku = ?`, sku).
		Scan(&current.ID, &current.SKU, &current.Name, &current.Quantity, &current.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Item{}, model.ErrNotFound
	}
	return current, err
}
