package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/gradis/ya-pr_shorturl/internal/config"
	database "github.com/gradis/ya-pr_shorturl/internal/config/db"
	"github.com/gradis/ya-pr_shorturl/internal/handler"
	"github.com/gradis/ya-pr_shorturl/internal/repository"
	"github.com/gradis/ya-pr_shorturl/internal/service"
)

type storage interface {
	service.URLRepository
	handler.DatabasePinger
	Close()
}

func createStorage(ctx context.Context, cfg config.Config) (storage, error) {
	if strings.TrimSpace(cfg.DatabaseDSN) != "" {
		if err := database.RunMigrations(cfg.DatabaseDSN, "migrations"); err != nil {
			return nil, fmt.Errorf("run PostgreSQL migrations: %w", err)
		}

		db, err := database.New(ctx, cfg.DatabaseDSN)
		if err != nil {
			return nil, fmt.Errorf("initialize PostgreSQL: %w", err)
		}

		return repository.NewPostgresRepository(db.Pool), nil
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

		return fileRepo, nil
	}

	return repository.NewMemoryRepository(), nil
}
