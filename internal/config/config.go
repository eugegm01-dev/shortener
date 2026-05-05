// Package config provides configuration loading from flags and environment variables.
package config

import (
	"flag"
	"os"
)

// Config holds all configuration parameters for the application.
type Config struct {
	ServerAddr      string // HTTP server address, e.g. ":8080"
	BaseURL         string // Public base URL for shortened links
	FileStoragePath string // Path to JSON file for file‑based storage (empty = disabled)
	DatabaseDSN     string // PostgreSQL DSN (empty = disabled)
	SecretKey       string // Secret key for cookie signing
	AuditFile       string // Path to audit log file (empty = disabled)
	AuditURL        string // URL of remote audit receiver (empty = disabled)
}

// LoadConfig reads configuration from command line flags and environment variables.
// Environment variables override flags:
//
//	SERVER_ADDRESS, BASE_URL, FILE_STORAGE_PATH, DATABASE_DSN, SECRET_KEY, AUDIT_FILE, AUDIT_URL.
func LoadConfig() *Config {
	cfg := &Config{
		ServerAddr:      ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "storage.json",
		SecretKey:       "default-secret-key",
		AuditFile:       "",
		AuditURL:        "",
	}

	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "Base URL for shortened links")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "File storage path (JSON)")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "Database DSN (PostgreSQL)")
	flag.StringVar(&cfg.SecretKey, "s", cfg.SecretKey, "Secret key for cookie signing")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "Path to audit log file")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "URL of remote audit receiver")

	flag.Parse()

	// Environment variables override
	if envAddr := os.Getenv("SERVER_ADDRESS"); envAddr != "" {
		cfg.ServerAddr = envAddr
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}
	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		cfg.FileStoragePath = envFilePath
	}
	if envDSN := os.Getenv("DATABASE_DSN"); envDSN != "" {
		cfg.DatabaseDSN = envDSN
	}
	if envSecret := os.Getenv("SECRET_KEY"); envSecret != "" {
		cfg.SecretKey = envSecret
	}
	if envAuditFile := os.Getenv("AUDIT_FILE"); envAuditFile != "" {
		cfg.AuditFile = envAuditFile
	}
	if envAuditURL := os.Getenv("AUDIT_URL"); envAuditURL != "" {
		cfg.AuditURL = envAuditURL
	}

	return cfg
}
