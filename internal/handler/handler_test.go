package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/storage"
)

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

    if rr2.Code != http.StatusConflict {   // было http.StatusCreated
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
