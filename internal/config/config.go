package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddr      string
	BaseURL         string
	FileStoragePath string
	DatabaseDSN     string // new field
}

func LoadConfig() *Config {
	cfg := &Config{
		ServerAddr:      ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "storage.json",
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "Base URL for shortened links")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "File storage path (JSON)")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "Database DSN (PostgreSQL)") // new flag

	flag.Parse()

	if envAddr := os.Getenv("SERVER_ADDRESS"); envAddr != "" {
		cfg.ServerAddr = envAddr
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}
	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		cfg.FileStoragePath = envFilePath
	}
	if envDSN := os.Getenv("DATABASE_DSN"); envDSN != "" { // new env
		cfg.DatabaseDSN = envDSN
	}

	return cfg
}
