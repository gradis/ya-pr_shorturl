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

func InitPostgresSchema(ctx context.Context, pool *pgxpool.Pool) error {
	const query = `
CREATE TABLE IF NOT EXISTS urls (
	id BIGSERIAL PRIMARY KEY,
	short_url VARCHAR(255) NOT NULL UNIQUE,
	original_url TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

	if _, err := pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("create urls table: %w", err)
	}

	const indexQuery = `CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_original_url ON urls (original_url);`

	if _, err := pool.Exec(ctx, indexQuery); err != nil {
		return fmt.Errorf("create original URL index: %w", err)
	}

	return nil
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
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

	result := make([]URLRecord, 0, len(records))
	for _, record := range records {
		var shortURL string
		if err := tx.QueryRow(ctx, query, record.ID, record.OriginalURL).Scan(&shortURL); err != nil {
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

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (string, error) {
	const query = `SELECT short_url, original_url FROM urls WHERE short_url = $1;`

	var shortURL string
	var originalURL string

	err := r.pool.QueryRow(ctx, query, id).Scan(&shortURL, &originalURL)
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

var _ URLRepository = (*PostgresRepository)(nil)
