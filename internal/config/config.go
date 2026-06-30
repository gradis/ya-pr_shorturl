package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
}

func Parse() *Config {
	cfg := &Config{
		ServerAddress:   "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "storage.json",
		DatabaseDSN:     "postgres://app:root@localhost:5432/short_url?sslmode=disable",
	}

	flag.StringVar(
		&cfg.ServerAddress,
		"a",
		cfg.ServerAddress,
		"HTTP server address")

	flag.StringVar(
		&cfg.BaseURL,
		"b",
		cfg.BaseURL,
		"base URL for shortened links",
	)
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "file storage path")

	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database DSN")

	flag.Parse()

	if v := os.Getenv("SERVER_ADDRESS"); v != "" {
		cfg.ServerAddress = v
	}

	if v := os.Getenv("BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	if v := os.Getenv("FILE_STORAGE_PATH"); v != "" {
		cfg.FileStoragePath = v
	}

	if v := os.Getenv("DATABASE_DSN"); v != "" {
		cfg.DatabaseDSN = v
	}

	return cfg
}
