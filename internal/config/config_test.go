package config

import (
	"flag"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Сохраняем оригинальные значения
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name     string
		args     []string
		envVars  map[string]string
		wantAddr string
		wantURL  string
	}{
		{
			name:     "default values",
			args:     []string{"test"},
			envVars:  map[string]string{},
			wantAddr: ":8080",
			wantURL:  "http://localhost:8080",
		},
		{
			name:     "with flags only",
			args:     []string{"test", "-a", ":9090", "-b", "http://example.com"},
			envVars:  map[string]string{},
			wantAddr: ":9090",
			wantURL:  "http://example.com",
		},
		{
			name: "env vars override defaults",
			args: []string{"test"},
			envVars: map[string]string{
				"SERVER_ADDRESS": ":8888",
				"BASE_URL":       "http://custom.com",
			},
			wantAddr: ":8888",
			wantURL:  "http://custom.com",
		},
		{
			name: "env vars override flags",
			args: []string{"test", "-a", ":9090", "-b", "http://example.com"},
			envVars: map[string]string{
				"SERVER_ADDRESS": ":7777",
				"BASE_URL":       "http://env.com",
			},
			wantAddr: ":7777",
			wantURL:  "http://env.com",
		},
		{
			name: "partial env vars with flags",
			args: []string{"test", "-a", ":9090"},
			envVars: map[string]string{
				"BASE_URL": "http://env-only.com",
			},
			wantAddr: ":9090",
			wantURL:  "http://env-only.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаем флаги
			os.Args = tt.args

			// Устанавливаем переменные окружения
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			// Сбрасываем флаги
			flag.CommandLine = flag.NewFlagSet(tt.args[0], flag.ExitOnError)

			cfg := LoadConfig()

			if cfg.ServerAddr != tt.wantAddr {
				t.Errorf("ServerAddr = %v, want %v", cfg.ServerAddr, tt.wantAddr)
			}
			if cfg.BaseURL != tt.wantURL {
				t.Errorf("BaseURL = %v, want %v", cfg.BaseURL, tt.wantURL)
			}
		})
	}
}
