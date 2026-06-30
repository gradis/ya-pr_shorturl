package repository

import (
	"context"
)

type URLRepository interface {
	SaveIfNotExist(ctx context.Context, id string, originalURL string) (bool, error)
	SaveBatch(ctx context.Context, records []URLRecord) error
	GetByID(ctx context.Context, id string) (string, error)
}

type URLRecord struct {
	ID          string
	OriginalURL string
}
