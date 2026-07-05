package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) SaveIfNotExist(ctx context.Context, id string, originalURL string) (bool, error) {
	const query = `INSERT INTO urls (short_url, original_url) VALUES ($1, $2) ON CONFLICT (short_url) DO NOTHING;`

	tag, err := r.pool.Exec(ctx, query, id, originalURL)
	if err != nil {
		return false, fmt.Errorf("insert URL: %w", err)
	}

	return tag.RowsAffected() == 1, nil
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

var _ URLRepository = (*PostgresRepository)(nil)
