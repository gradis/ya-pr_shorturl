package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/gradis/ya-pr_shorturl/internal/config"
	database "github.com/gradis/ya-pr_shorturl/internal/config/db"
	"github.com/gradis/ya-pr_shorturl/internal/handler"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
)

type storageDependencies struct {
	repository repository.URLRepository
	pinger     handler.DatabasePinger
	close      func()
}

func createStorage(ctx context.Context, cfg config.Config) (*storageDependencies, error) {
	if strings.TrimSpace(cfg.DatabaseDSN) != "" {
		db, err := database.New(ctx, cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("initialize PostgreSQL: %w", err)
		}

		if err := repository.InitPostgresSchema(ctx, db.Pool); err != nil {
			db.Close()
			return nil, fmt.Errorf("initialize PostgreSQL schema: %w", err)
		}

		return &storageDependencies{
			repository: repository.NewPostgresRepository(db.Pool),
			pinger:     db.Pool,
			close:      db.Close,
		}, nil
	}

	if strings.TrimSpace(cfg.FileStoragePath) != "" {
		fileRepo, err := repository.NewFileRepository(
			cfg.FileStoragePath,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"initialize file repository: %w",
				err,
			)
		}

		return &storageDependencies{
			repository: fileRepo,
			pinger:     nil,
			close:      func() {},
		}, nil
	}

	return &storageDependencies{
		repository: repository.NewMemoryRepository(),
		pinger:     nil,
		close:      func() {},
	}, nil
}
