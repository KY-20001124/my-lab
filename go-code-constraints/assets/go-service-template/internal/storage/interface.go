package storage

import (
	"context"
	"errors"

	"example.com/go-service-template/pkg/types"
)

var ErrNotFound = errors.New("not found")

type ThingStore interface {
	Get(ctx context.Context, id string) (*types.Thing, error)
}
