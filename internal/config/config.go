package config

import (
	"flag"
	"os"
	"strings"
)

type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string
	AuditFile       string
	AuditURL        string
	AuthSecret      string
}

func Parse() *Config {
	cfg := &Config{
		ServerAddress:   "localhost:8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "",
		DatabaseDSN:     "",
		AuthSecret:      "shortener-auth-secret",
	}

	flag.StringVar(
		&cfg.ServerAddress,
		"a",
		cfg.ServerAddress,
		"server address",
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

	flag.StringVar(
		&cfg.AuthSecret,
		"s",
		cfg.AuthSecret,
		"authentication cookie signing secret",
	)

	flag.StringVar(&cfg.AuditFile, "audit-file", "", "audit log file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "audit receiver URL")

	flag.Parse()

	if value, ok := lookupNonEmptyEnv("SERVER_ADDRESS"); ok {
		cfg.ServerAddress = value
	}

	if value, ok := lookupNonEmptyEnv("BASE_URL"); ok {
		cfg.BaseURL = value
	}

	if value, ok := lookupNonEmptyEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = value
	}

	if value, ok := lookupNonEmptyEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = value
	}

	if value, ok := lookupNonEmptyEnv("AUTH_SECRET"); ok {
		cfg.AuthSecret = value
	}

	if value, ok := lookupNonEmptyEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = value
	}
	if value, ok := lookupNonEmptyEnv("AUDIT_URL"); ok {
		cfg.AuditURL = value
	}

	return cfg
}

func lookupNonEmptyEnv(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", false
	}

	return value, true
}
