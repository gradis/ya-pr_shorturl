package config

import "flag"

type Config struct {
	ServerAddress string
	BaseURL       string
}

func Parse() *Config {
	cfg := &Config{}

	flag.StringVar(
		&cfg.ServerAddress,
		"a",
		"localhost:8080",
		"HTTP server address")

	flag.StringVar(
		&cfg.BaseURL,
		"b",
		"http://localhost:8080",
		"base URL for shortened links",
	)
	flag.Parse()

	return cfg
}
