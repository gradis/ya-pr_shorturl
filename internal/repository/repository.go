package repository

import (
	"context"
)

type URLRepository interface {
	SaveIfNotExist(ctx context.Context, id string, originalURL string) (bool, error)
	GetById(ctx context.Context, id string) (string, error)
}
