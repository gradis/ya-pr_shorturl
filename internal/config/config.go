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
		FileStoragePath: "",
		DatabaseDSN:     "",
	}

	flag.StringVar(
		&cfg.ServerAddress,
		"a",
		cfg.ServerAddress,
		"HTTP server address",
	)

	flag.StringVar(
		&cfg.BaseURL,
		"b",
		cfg.BaseURL,
		"base URL",
	)

	flag.StringVar(
		&cfg.FileStoragePath,
		"f",
		cfg.FileStoragePath,
		"file storage path",
	)

	flag.StringVar(
		&cfg.DatabaseDSN,
		"d",
		cfg.DatabaseDSN,
		"database DSN",
	)

	flag.Parse()

	if value, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.ServerAddress = value
	}

	if value, ok := os.LookupEnv("BASE_URL"); ok {
		cfg.BaseURL = value
	}

	if value, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = value
	}

	if value, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = value
	}

	return cfg
}
