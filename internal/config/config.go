package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress string
	BaseURL       string
}

func Parse() *Config {
	cfg := &Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
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
	flag.Parse()

	if v := os.Getenv("SERVER_ADDRESS"); v != "" {
		cfg.ServerAddress = v
	}

	if v := os.Getenv("BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	return cfg
}
