package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_ServeHTTP_PostSuccess(t *testing.T) {
	store := NewURLStore()
	h := &handler{store: store}

	reqBody := "https://example.com"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	body := w.Body.String()
	if !strings.HasPrefix(body, "http://localhost:8080/") {
		t.Errorf("Expected short URL, got %s", body)
	}
	if len(body) <= len("http://localhost:8080/") {
		t.Errorf("Short URL is too short")
	}
}

func TestHandler_ServeHTTP_PostEmptyBody(t *testing.T) {
	store := NewURLStore()
	h := &handler{store: store}

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	req.Header.Set("Content-Type", "text/plain")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandler_ServeHTTP_GetSuccess(t *testing.T) {
	store := NewURLStore()
	h := &handler{store: store}

	// Сначала создаем короткий URL
	originalURL := "https://example.com"
	shortID := store.Save(originalURL)

	req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected status %d, got %d", http.StatusTemporaryRedirect, w.Code)
	}

	location := w.Header().Get("Location")
	if location != originalURL {
		t.Errorf("Expected Location %s, got %s", originalURL, location)
	}
}

func TestHandler_ServeHTTP_GetNotFound(t *testing.T) {
	store := NewURLStore()
	h := &handler{store: store}

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandler_ServeHTTP_InvalidMethod(t *testing.T) {
	store := NewURLStore()
	h := &handler{store: store}

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandler_ServeHTTP_InvalidPath(t *testing.T) {
	store := NewURLStore()
	h := &handler{store: store}

	// Путь "/" с методом GET - это некорректно, потому что длина пути не больше 1
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
