package config

import (
	"flag"
	"os"
)

// Config содержит конфигурацию приложения
type Config struct {
	ServerAddr string
	BaseURL    string
}

// LoadConfig загружает конфигурацию из флагов и переменных окружения
func LoadConfig() *Config {
	cfg := &Config{
		ServerAddr: ":8080",
		BaseURL:    "http://localhost:8080",
	}

	// Чтение флагов
	flag.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", cfg.BaseURL, "Base URL for shortened links")
	flag.Parse()

	// Переопределение переменными окружения
	if envAddr := os.Getenv("SERVER_ADDRESS"); envAddr != "" {
		cfg.ServerAddr = envAddr
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}

	return cfg
}
