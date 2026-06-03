package mysql

import (
	"context"
	"errors"
	"fmt"

	"example.com/go-service-template/internal/storage"
	"example.com/go-service-template/pkg/types"
)

type ThingTable struct {
	ID   string `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name"`
}

type ThingStore struct {
	rows map[string]ThingTable
}

func NewThingStore() storage.ThingStore {
	return &ThingStore{
		rows: map[string]ThingTable{
			"demo": {ID: "demo", Name: "Demo Thing"},
		},
	}
}

func (s *ThingStore) Get(ctx context.Context, id string) (*types.Thing, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("get thing canceled: %w", ctx.Err())
	default:
	}
	row, ok := s.rows[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return thingFromTable(row), nil
}

func thingFromTable(row ThingTable) *types.Thing {
	return &types.Thing{
		ID:   row.ID,
		Name: row.Name,
	}
}

var _ storage.ThingStore = (*ThingStore)(nil)

var _ = errors.Is
