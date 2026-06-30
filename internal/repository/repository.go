package repository

import (
	"context"
)

type URLRepository interface {
	SaveURL(ctx context.Context, id string, originalURL string) (URLSaveResult, error)
	SaveBatch(ctx context.Context, records []URLRecord) ([]URLRecord, error)
	GetByID(ctx context.Context, id string) (string, error)
}

type URLRecord struct {
	ID          string
	OriginalURL string
}

type URLSaveResult struct {
	ID        string
	Duplicate bool
}
