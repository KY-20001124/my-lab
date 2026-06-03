package service

import (
	"context"
	"errors"
	"fmt"

	"example.com/go-service-template/internal/storage"
	"example.com/go-service-template/pkg/types"
)

var ErrThingNotFound = errors.New("thing not found")

type ThingService interface {
	Get(ctx context.Context, id string) (*types.Thing, error)
}

type thingService struct {
	store storage.ThingStore
}

func NewThingService(store storage.ThingStore) ThingService {
	if store == nil {
		panic("thing store is nil")
	}
	return &thingService{store: store}
}

func (s *thingService) Get(ctx context.Context, id string) (*types.Thing, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	thing, err := s.store.Get(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, fmt.Errorf("%w: id=%s", ErrThingNotFound, id)
		}
		return nil, fmt.Errorf("get thing: %w", err)
	}
	return thing, nil
}

var _ ThingService = (*thingService)(nil)
