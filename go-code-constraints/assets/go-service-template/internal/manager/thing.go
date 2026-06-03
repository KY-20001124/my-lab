package manager

import (
	"context"

	"example.com/go-service-template/internal/service"
	"example.com/go-service-template/pkg/types"
)

type ThingManager interface {
	Get(ctx context.Context, id string) (*types.Thing, error)
}

type thingManager struct {
	things service.ThingService
}

func NewThingManager(things service.ThingService) ThingManager {
	return &thingManager{things: things}
}

func (m *thingManager) Get(ctx context.Context, id string) (*types.Thing, error) {
	return m.things.Get(ctx, id)
}
