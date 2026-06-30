package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(databaseDSN string) error {
	migrationPath, err := findMigrationsDir()
	if err != nil {
		return fmt.Errorf("failed to find migrations directory: %w", err)
	}

	sourceURL := "file://" + filepath.ToSlash(migrationPath)

	migrator, err := migrate.New(
		sourceURL,
		databaseDSN,
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	defer func() {
		_, _ = migrator.Close()
	}()

	if err := migrator.Up(); err != nil &&
		!errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

func findMigrationsDir() (string, error) {
	candidates := make([]string, 0, 4)

	if workingDir, err := os.Getwd(); err != nil {
		candidates = append(candidates, filepath.Join(workingDir, "migrations"))
	}

	if executablePath, err := os.Executable(); err != nil {
		executableDir := filepath.Dir(executablePath)

		candidates = append(candidates, filepath.Join(executableDir, "migrations"))
	}

	if _, currentFile, _, ok := runtime.Caller(0); ok {
		projectRoot := filepath.Clean(
			filepath.Join(filepath.Dir(currentFile), "../../.."),
		)

		candidates = append(
			candidates,
			filepath.Join(projectRoot, "migrations"),
		)
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}

		if info.IsDir() {
			absolutePath, err := filepath.Abs(candidate)
			if err != nil {
				return "", fmt.Errorf(
					"resolve absolute migrations path %q: %w",
					candidate,
					err,
				)
			}

			return absolutePath, nil
		}
	}

	return "", fmt.Errorf(
		"migrations directory not found; checked: %v",
		candidates,
	)
}
