package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/gradis/ya-pr_shorturl/internal/auth"
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
INSERT INTO urls (short_url, original_url, user_id)
VALUES ($1, $2, $3)
ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url
RETURNING short_url, (xmax = 0) AS inserted;`

	var shortURL string
	var inserted bool
	userID, _ := auth.UserIDFromContext(ctx)

	if err := r.pool.QueryRow(ctx, query, id, originalURL, userID).Scan(&shortURL, &inserted); err != nil {
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
INSERT INTO urls (short_url, original_url, user_id)
VALUES ($1, $2, $3)
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
	userID, _ := auth.UserIDFromContext(ctx)
	for _, record := range records {
		recordUserID := record.UserID
		if recordUserID == "" {
			recordUserID = userID
		}
		batch.Queue(query, record.ID, record.OriginalURL, recordUserID)
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
			UserID:      userID,
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

func (r *PostgresRepository) GetByUserID(ctx context.Context, userID string) ([]URLRecord, error) {
	const query = `
SELECT short_url, original_url, user_id
FROM urls
WHERE user_id = $1
ORDER BY id;`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("select user URLs: %w", err)
	}
	defer rows.Close()

	records := make([]URLRecord, 0)
	for rows.Next() {
		var record URLRecord
		if err := rows.Scan(&record.ID, &record.OriginalURL, &record.UserID); err != nil {
			return nil, fmt.Errorf("scan user URL: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user URLs: %w", err)
	}

	return records, nil
}

func (r *PostgresRepository) GetByID(
	ctx context.Context,
	id string,
) (string, error) {
	const query = `
SELECT original_url, is_deleted
FROM urls
WHERE short_url = $1;
`

	var originalURL string
	var isDeleted bool

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&originalURL,
		&isDeleted,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrURLNotFound
		}

		return "", fmt.Errorf("select URL: %w", err)
	}

	if isDeleted {
		return "", ErrURLDeleted
	}

	return originalURL, nil
}

func (r *PostgresRepository) DeleteBatch(
	ctx context.Context,
	records []URLDeleteRecord,
) error {
	if len(records) == 0 {
		return nil
	}

	shortURLs := make([]string, 0, len(records))
	userIDs := make([]string, 0, len(records))

	for _, record := range records {
		shortURLs = append(shortURLs, record.ID)
		userIDs = append(userIDs, record.UserID)
	}

	const query = `
	UPDATE urls AS u
	SET is_deleted = TRUE
	FROM unnest($1::text[], $2::text[]) AS deleted(short_url, user_id)
	WHERE u.short_url = deleted.short_url
	  AND u.user_id = deleted.user_id;
	`

	if _, err := r.pool.Exec(ctx, query, shortURLs, userIDs); err != nil {
		return fmt.Errorf("mark URLs as deleted: %w", err)
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
