package repository

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/junnotantra/backend-test/internal/model"
	_ "modernc.org/sqlite"
)

func TestAdjustStockConcurrentPreventsOverselling(t *testing.T) {
	repo, err := Open(filepath.Join(t.TempDir(), "inventory.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	if _, err := repo.DB().Exec(`
		CREATE TABLE items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sku TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE inventory (
			item_id INTEGER PRIMARY KEY,
			quantity INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (item_id) REFERENCES items(id)
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	item, err := repo.CreateItem(context.Background(), "SKU-001", "Test Item", 10)
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	const requestCount = 2
	start := make(chan struct{})
	results := make(chan error, requestCount)

	var wg sync.WaitGroup
	wg.Add(requestCount)
	for range requestCount {
		go func() {
			defer wg.Done()
			<-start

			_, err := repo.AdjustStock(context.Background(), item.ID, -10)
			results <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var successCount, negativeStockCount int
	for err := range results {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, model.ErrNegativeStock):
			negativeStockCount++
		default:
			t.Errorf("unexpected adjustment error: %v", err)
		}
	}

	if successCount != 1 {
		t.Errorf("successful adjustments = %d, want 1", successCount)
	}
	if negativeStockCount != 1 {
		t.Errorf("negative-stock rejections = %d, want 1", negativeStockCount)
	}

	updated, err := repo.GetItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get adjusted item: %v", err)
	}
	if updated.Quantity != 0 {
		t.Errorf("final quantity = %d, want 0", updated.Quantity)
	}
}
