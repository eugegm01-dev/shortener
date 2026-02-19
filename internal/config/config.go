package config

import (
	"flag"
	"os"
)

// Config contains application configuration
type Config struct {
	ServerAddr      string
	BaseURL         string
	FileStoragePath string // new field
}

// LoadConfig loads configuration from flags and environment variables
func LoadConfig() *Config {
	cfg := &Config{
		ServerAddr:      ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "storage.json", // default file name
	}

	// Define flags
	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "Base URL for shortened links")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "File storage path (JSON)")

	flag.Parse()

	// Override with environment variables
	if envAddr := os.Getenv("SERVER_ADDRESS"); envAddr != "" {
		cfg.ServerAddr = envAddr
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}
	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		cfg.FileStoragePath = envFilePath
	}

	return cfg
}
