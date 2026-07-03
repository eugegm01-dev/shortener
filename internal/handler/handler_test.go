package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eugegm01-dev/shortener/internal/auth"
	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/service"
	"github.com/eugegm01-dev/shortener/internal/storage"
	"github.com/go-chi/chi/v5"
)

func addUserIDToContext(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), userIDKey, userID)
	return r.WithContext(ctx)
}

func TestShortenURLJSON(t *testing.T) {
	store := storage.NewMemoryStorage()
	defer store.Close()
	cfg := &config.Config{
		ServerAddr: ":8080",
		BaseURL:    "http://localhost:8080",
	}
	deleter := service.NewDeleter(store, 100, 5*time.Second)
	defer deleter.Close()
	h := New(store, cfg, deleter)

	tests := []struct {
		name          string
		requestBody   string
		expectedCode  int
		checkResponse func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name:         "valid URL",
			requestBody:  `{"url": "https://practicum.yandex.ru"}`,
			expectedCode: http.StatusCreated,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var result map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if result["result"] == "" {
					t.Error("Expected non-empty result URL")
				}
			},
		},
		{
			name:         "empty URL",
			requestBody:  `{"url": ""}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid JSON",
			requestBody:  `invalid json`,
			expectedCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			h.ShortenURLJSON(rr, req)
			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestShortenURLJSON_DuplicateURL(t *testing.T) {
	store := storage.NewMemoryStorage()
	defer store.Close()
	cfg := &config.Config{ServerAddr: ":8080", BaseURL: "http://localhost:8080"}
	deleter := service.NewDeleter(store, 100, 5*time.Second)
	defer deleter.Close()
	h := New(store, cfg, deleter)

	requestBody := `{"url": "https://example.com"}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(requestBody))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	h.ShortenURLJSON(rr1, req1)
	if rr1.Code != http.StatusCreated {
		t.Fatalf("First request failed with status %d", rr1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(requestBody))
	rr2 := httptest.NewRecorder()
	h.ShortenURLJSON(rr2, req2)
	if rr2.Code != http.StatusConflict {
		t.Fatalf("Second request expected status %d, got %d", http.StatusConflict, rr2.Code)
	}
}

func TestHandlerPing(t *testing.T) {
	store := storage.NewMemoryStorage()
	defer store.Close()
	cfg := &config.Config{ServerAddr: ":8080", BaseURL: "http://localhost:8080"}
	deleter := service.NewDeleter(store, 100, 5*time.Second)
	defer deleter.Close()
	h := New(store, cfg, deleter)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()
	h.Ping(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}
}

func TestHandler_GetUserURLs(t *testing.T) {
	store := storage.NewMemoryStorage()
	cfg := &config.Config{BaseURL: "http://localhost:8080", SecretKey: "test-secret"}
	deleter := service.NewDeleter(store, 100, 5*time.Second)
	defer deleter.Close()
	h := New(store, cfg, deleter)

	userID := "test-user"
	store.SaveWithUser("https://example.com/1", userID)
	store.SaveWithUser("https://example.com/2", userID)
	cookie, _ := auth.SignCookie(userID, cfg.SecretKey)

	req := httptest.NewRequest("GET", "/api/user/urls", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	h.GetUserURLs(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}
	var resp []map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(resp))
	}
}

func TestHandler_DeleteUserURLs(t *testing.T) {
	store := storage.NewMemoryStorage()
	cfg := &config.Config{BaseURL: "http://localhost:8080", SecretKey: "test-secret"}

	// ВАЖНО: Для теста создаем "быстрый" делетер.
	// batchSize = 1, timeout = 5ms. Это заставит воркер сбрасывать буфер почти мгновенно.
	deleter := service.NewDeleter(store, 1, 5*time.Millisecond)
	defer deleter.Close()

	h := New(store, cfg, deleter)
	userID := "test-user"
	id1, _, _ := store.SaveWithUser("https://example.com/1", userID)
	id2, _, _ := store.SaveWithUser("https://example.com/2", userID)

	body := bytes.NewBufferString(fmt.Sprintf(`["%s", "%s"]`, id1, id2))
	req := httptest.NewRequest("DELETE", "/api/user/urls", body)
	req = addUserIDToContext(req, userID)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h.DeleteUserURLs(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("Expected status Accepted, got %d", rr.Code)
	}

	// Даём фоновому воркеру время на flush (50мс с запасом)
	time.Sleep(50 * time.Millisecond)

	_, err1 := store.Get(id1)
	_, err2 := store.Get(id2)
	if err1 != storage.ErrGone || err2 != storage.ErrGone {
		t.Error("URLs should be marked as deleted")
	}
}

func TestHandler_RedirectURL(t *testing.T) {
	store := storage.NewMemoryStorage()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	deleter := service.NewDeleter(store, 100, 5*time.Second)
	defer deleter.Close()
	h := New(store, cfg, deleter)

	url := "https://redirect-test.com"
	id, _, _ := store.Save(url)
	req := httptest.NewRequest("GET", "/"+id, nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
		URLParams: chi.RouteParams{Keys: []string{"id"}, Values: []string{id}},
	}))

	rr := httptest.NewRecorder()
	h.RedirectURL(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected 307, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != url {
		t.Errorf("Expected Location %s, got %s", url, loc)
	}
}
