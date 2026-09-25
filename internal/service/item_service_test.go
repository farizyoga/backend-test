package service

import (
	"context"
	"testing"

	"github.com/junnotantra/backend-test/internal/model"
)

type fakeRepository struct{ called bool }

func (f *fakeRepository) CreateItem(context.Context, string, string, int64) (model.Item, error) {
	f.called = true
	return model.Item{SKU: "KB-001"}, nil
}
func (*fakeRepository) ListItems(context.Context) ([]model.Item, error)    { return nil, nil }
func (*fakeRepository) GetItem(context.Context, int64) (model.Item, error) { return model.Item{}, nil }
func (*fakeRepository) AdjustStock(context.Context, int64, int64) (model.Item, error) {
	return model.Item{}, nil
}
func (*fakeRepository) GetItemBySKU(context.Context, string) (model.Item, error) {
	return model.Item{}, model.ErrNotFound
}

func TestCreateItemUsesInjectedRepository(t *testing.T) {
	repo := &fakeRepository{}
	got, err := NewItemService(repo).CreateItem(context.Background(), "KB-001", "Keyboard", 10)
	if err != nil || got.SKU != "KB-001" || !repo.called {
		t.Fatalf("item=%+v err=%v called=%v", got, err, repo.called)
	}
}

func TestCreateItemValidatesInput(t *testing.T) {
	repo := &fakeRepository{}
	if _, err := NewItemService(repo).CreateItem(context.Background(), "", "Keyboard", 10); err != ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if repo.called {
		t.Fatal("repository should not be called for invalid input")
	}
}
