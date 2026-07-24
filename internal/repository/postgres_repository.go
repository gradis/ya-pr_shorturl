package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *PostgresRepository) Close() {
	r.pool.Close()
}

func (r *PostgresRepository) SaveURL(ctx context.Context, id string, originalURL string) (URLSaveResult, error) {
	const query = `
INSERT INTO urls (short_url, original_url)
VALUES ($1, $2)
ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url
RETURNING short_url, (xmax = 0) AS inserted;`

	var shortURL string
	var inserted bool

	if err := r.pool.QueryRow(ctx, query, id, originalURL).Scan(&shortURL, &inserted); err != nil {
		if isUniqueViolation(err) {
			return URLSaveResult{}, ErrURLIDConflict
		}

		return URLSaveResult{}, fmt.Errorf("insert URL: %w", err)
	}

	return URLSaveResult{
		ID:        shortURL,
		Duplicate: !inserted,
	}, nil
}

func (r *PostgresRepository) SaveBatch(ctx context.Context, records []URLRecord) ([]URLRecord, error) {
	const query = `
INSERT INTO urls (short_url, original_url)
VALUES ($1, $2)
ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url
RETURNING short_url;`

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	batch := &pgx.Batch{}
	for _, record := range records {
		batch.Queue(query, record.ID, record.OriginalURL)
	}

	batchResults := tx.SendBatch(ctx, batch)
	result := make([]URLRecord, 0, len(records))
	for _, record := range records {
		var shortURL string
		if err := batchResults.QueryRow().Scan(&shortURL); err != nil {
			_ = batchResults.Close()
			if isUniqueViolation(err) {
				return nil, ErrURLIDConflict
			}

			return nil, fmt.Errorf("insert URL: %w", err)
		}

		result = append(result, URLRecord{
			ID:          shortURL,
			OriginalURL: record.OriginalURL,
		})
	}

	if err := batchResults.Close(); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrURLIDConflict
		}

		return nil, fmt.Errorf("close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (string, error) {
	const query = `SELECT original_url FROM urls WHERE short_url = $1;`

	var originalURL string

	err := r.pool.QueryRow(ctx, query, id).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrURLNotFound
		}

		return "", fmt.Errorf("select URL: %w", err)
	}

	return originalURL, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
