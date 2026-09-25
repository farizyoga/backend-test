package repository

import (
	"context"

	"github.com/junnotantra/backend-test/internal/model"
)

type Repository interface {
	CreateItem(context.Context, string, string, int64) (model.Item, error)
	ListItems(context.Context) ([]model.Item, error)
	GetItem(context.Context, int64) (model.Item, error)
	AdjustStock(context.Context, int64, int64) (model.Item, error)
	GetItemBySKU(context.Context, string) (model.Item, error)
}
