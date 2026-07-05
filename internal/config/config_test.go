package config

import "testing"

func TestParseIgnoresDatabaseConnString(t *testing.T) {
	t.Setenv("DATABASE_CONN_STRING", "postgres://app:root@localhost:5432/shortener?sslmode=disable")
	t.Setenv("FILE_STORAGE_PATH", "/tmp/storage.json")

	cfg := Parse()

	if cfg.DatabaseDSN != "" {
		t.Fatalf("expected empty DatabaseDSN, got %q", cfg.DatabaseDSN)
	}

	if cfg.FileStoragePath != "/tmp/storage.json" {
		t.Fatalf("expected file storage path, got %q", cfg.FileStoragePath)
	}
}
