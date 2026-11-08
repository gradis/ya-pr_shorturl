package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// Postgres содержит пул соединений PostgreSQL.
type Postgres struct {
	// Pool — пул соединений, закрываемый методом Close.
	Pool *pgxpool.Pool
}

// New создаёт пул по DSN, не проверяя соединение с сервером.
// Для проверки доступности вызовите Pool.Ping; после работы вызовите Close.
func New(ctx context.Context, dsn string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	return &Postgres{Pool: pool}, nil
}

// RunMigrations применяет неприменённые миграции из локального каталога.
// Отсутствие новых миграций не считается ошибкой.
func RunMigrations(dsn string, migrationsPath string) error {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parse postgres config: %w", err)
	}

	sqlDB := stdlib.OpenDB(*cfg.ConnConfig)
	defer sqlDB.Close()

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create postgres migration driver: %w", err)
	}

	migration, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("initialize migrations: %w", err)
	}

	if err := migration.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// Close освобождает пул соединений; допустим вызов для nil получателя.
func (p *Postgres) Close() {
	if p != nil && p.Pool != nil {
		p.Pool.Close()
	}
}
