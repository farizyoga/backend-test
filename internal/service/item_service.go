package service

import (
	"context"
	"errors"

	"github.com/junnotantra/backend-test/internal/model"
	"github.com/junnotantra/backend-test/internal/repository"
)

var ErrInvalidInput = errors.New("invalid item input")
var ErrSKUAlreadyExists = errors.New("sku already exists")

type ItemService struct{ repo repository.Repository }

func NewItemService(repo repository.Repository) *ItemService { return &ItemService{repo: repo} }

func (s *ItemService) CreateItem(ctx context.Context, sku, name string, quantity int64) (model.Item, error) {
	if sku == "" || name == "" || quantity < 0 {
		return model.Item{}, ErrInvalidInput
	}

	_, err := s.repo.GetItemBySKU(ctx, sku)
	if errors.Is(err, model.ErrNotFound) {
		return s.repo.CreateItem(ctx, sku, name, quantity)
	}

	if err != nil {
		return model.Item{}, err
	}

	return model.Item{}, ErrSKUAlreadyExists
}

func (s *ItemService) ListItems(ctx context.Context) ([]model.Item, error) {
	return s.repo.ListItems(ctx)
}

func (s *ItemService) GetItem(ctx context.Context, id int64) (model.Item, error) {
	return s.repo.GetItem(ctx, id)
}

func (s *ItemService) AdjustStock(ctx context.Context, id, delta int64) (model.Item, error) {
	if id < 1 {
		return model.Item{}, ErrInvalidInput
	}
	return s.repo.AdjustStock(ctx, id, delta)
}
