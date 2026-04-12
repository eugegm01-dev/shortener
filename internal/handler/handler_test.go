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
	"github.com/eugegm01-dev/shortener/internal/storage"
	"github.com/go-chi/chi/v5"
)

func addUserIDToContext(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), userIDKey, userID)
	return r.WithContext(ctx)
}
func TestShortenURLJSON(t *testing.T) {
	// Создаем хранилище и обработчик
	store := storage.NewMemoryStorage()
	defer store.Close()

	cfg := &config.Config{
		ServerAddr: ":8080",
		BaseURL:    "http://localhost:8080",
	}

	h := New(store, cfg)

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

				contentType := resp.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", contentType)
				}
			},
		},
		{
			name:         "empty URL",
			requestBody:  `{"url": ""}`,
			expectedCode: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var errResp map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}

				if errResp["error"] == "" {
					t.Error("Expected error message in response")
				}
			},
		},
		{
			name:         "invalid JSON",
			requestBody:  `invalid json`,
			expectedCode: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var errResp map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}

				if errResp["error"] == "" {
					t.Error("Expected error message in response")
				}
			},
		},
		{
			name:         "missing URL field",
			requestBody:  `{}`,
			expectedCode: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var errResp map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}

				if errResp["error"] == "" {
					t.Error("Expected error message in response")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем запрос
			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Создаем записыватель ответа
			rr := httptest.NewRecorder()

			// Вызываем обработчик
			h.ShortenURLJSON(rr, req)

			// Проверяем код статуса
			if rr.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, rr.Code)
			}

			// Проверяем ответ
			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}

func TestShortenURLJSON_DuplicateURL(t *testing.T) {
	store := storage.NewMemoryStorage()
	defer store.Close()

	cfg := &config.Config{
		ServerAddr: ":8080",
		BaseURL:    "http://localhost:8080",
	}

	h := New(store, cfg)

	// Первый запрос с одним и тем же URL
	requestBody := `{"url": "https://example.com"}`
	req1 := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(requestBody))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	h.ShortenURLJSON(rr1, req1)

	if rr1.Code != http.StatusCreated {
		t.Fatalf("First request failed with status %d", rr1.Code)
	}

	var resp1 map[string]string
	if err := json.NewDecoder(rr1.Body).Decode(&resp1); err != nil {
		t.Fatalf("Failed to decode first response: %v", err)
	}

	// Второй запрос с тем же URL
	req2 := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(requestBody))
	rr2 := httptest.NewRecorder()
	h.ShortenURLJSON(rr2, req2)

	if rr2.Code != http.StatusConflict { // было http.StatusCreated
		t.Fatalf("Second request expected status %d, got %d", http.StatusConflict, rr2.Code)
	}

	var resp2 map[string]string
	if err := json.NewDecoder(rr2.Body).Decode(&resp2); err != nil {
		t.Fatalf("Failed to decode second response: %v", err)
	}

	// Оба должны быть валидными короткими ссылками
	if resp2["result"] == "" {
		t.Error("Expected non-empty result URL for second request")
	}
}

// Вместо него добавьте простой тест

func TestHandlerPing(t *testing.T) {
	store := storage.NewMemoryStorage()
	defer store.Close()

	cfg := &config.Config{
		ServerAddr: ":8080",
		BaseURL:    "http://localhost:8080",
	}

	h := New(store, cfg)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()

	h.Ping(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "OK" {
		t.Errorf("Expected status 'OK', got %s", resp["status"])
	}
}
func TestHandler_GetUserURLs(t *testing.T) {
	store := storage.NewMemoryStorage()
	cfg := &config.Config{BaseURL: "http://localhost:8080", SecretKey: "test-secret"}
	h := New(store, cfg)

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
	h := New(store, cfg)

	userID := "test-user"
	id1, _, _ := store.SaveWithUser("https://example.com/1", userID)
	id2, _, _ := store.SaveWithUser("https://example.com/2", userID)

	body := bytes.NewBufferString(fmt.Sprintf(`["%s", "%s"]`, id1, id2))
	req := httptest.NewRequest("DELETE", "/api/user/urls", body)
	req = addUserIDToContext(req, userID) // ← используем контекст вместо cookie
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.DeleteUserURLs(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("Expected status Accepted, got %d", rr.Code)
	}
	time.Sleep(10 * time.Millisecond)

	_, err1 := store.Get(id1)
	_, err2 := store.Get(id2)
	if err1 != storage.ErrGone || err2 != storage.ErrGone {
		t.Error("URLs should be marked as deleted")
	}
}
func TestHandler_RedirectURL(t *testing.T) {
	store := storage.NewMemoryStorage()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	h := New(store, cfg)

	url := "https://redirect-test.com"
	id, _, _ := store.Save(url)

	req := httptest.NewRequest("GET", "/"+id, nil)
	// Имитируем параметр chi URL
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
