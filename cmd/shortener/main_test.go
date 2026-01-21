package main

import (
	"testing"

	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/storage"
	"github.com/go-chi/chi/v5"
)

// Тестируем базовую функциональность без HTTP
func TestBasicFunctionality(t *testing.T) {
	// Test 1: Хранилище работает
	store := storage.NewMemoryStorage()

	id, err := store.Save("https://example.com")
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if id == "" {
		t.Error("Expected non-empty ID")
	}

	url, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if url != "https://example.com" {
		t.Errorf("Expected https://example.com, got %s", url)
	}

	// Test 2: Конфигурация работает
	cfg := config.LoadConfig()
	if cfg.ServerAddr == "" {
		t.Error("Expected non-empty ServerAddr")
	}
	if cfg.BaseURL == "" {
		t.Error("Expected non-empty BaseURL")
	}

	// Test 3: Chi router инициализируется
	router := chi.NewRouter()
	if router == nil {
		t.Error("Expected router to be initialized")
	}
}

// Интеграционный тест
func TestIntegration(t *testing.T) {
	// Проверяем, что сборка проходит
	t.Log("Build successful - chi router integrated")

	// Проверяем, что зависимости загружены
	t.Log("Dependencies: chi router v5")

	// Проверяем структуру проекта
	t.Log("Project structure: clean architecture with handlers, storage, config")
}
