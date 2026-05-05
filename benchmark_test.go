package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/handler"
	"github.com/eugegm01-dev/shortener/internal/models"
	"github.com/eugegm01-dev/shortener/internal/storage"
)

func BenchmarkMemoryStorageSave(b *testing.B) {
	store := storage.NewMemoryStorage()
	url := "https://example.com/benchmark"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = store.Save(url)
	}
}

func BenchmarkMemoryStorageGet(b *testing.B) {
	store := storage.NewMemoryStorage()
	url := "https://example.com/benchmark"
	id, _, _ := store.Save(url)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Get(id)
	}
}

func BenchmarkShortenURLJSON(b *testing.B) {
	store := storage.NewMemoryStorage()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	h := handler.New(store, cfg)

	reqBody := models.ShortenRequest{URL: "https://practicum.yandex.ru"}
	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		h.ShortenURLJSON(rr, req)
	}
}

func BenchmarkRedirectURL(b *testing.B) {
	store := storage.NewMemoryStorage()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	h := handler.New(store, cfg)

	url := "https://example.com/redirect"
	id, _, _ := store.Save(url)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/"+id, nil)
		rr := httptest.NewRecorder()
		// Имитируем chi роутер, передавая параметр в контекст
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
			URLParams: chi.RouteParams{Keys: []string{"id"}, Values: []string{id}},
		}))
		h.RedirectURL(rr, req)
	}
}

func BenchmarkGenerateShortID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = storage.GenerateShortID()
	}
}
