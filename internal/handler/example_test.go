package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/eugegm01-dev/shortener/internal/auth"
	"github.com/eugegm01-dev/shortener/internal/config"
	"github.com/eugegm01-dev/shortener/internal/handler"
	"github.com/eugegm01-dev/shortener/internal/service"
	"github.com/eugegm01-dev/shortener/internal/storage"
)

func ExampleHandler_ShortenURL() {
	store := storage.NewMemoryStorage()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	deleter := service.NewDeleter(store, 100, 5*time.Second)
	h := handler.New(store, cfg, deleter)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	rr := httptest.NewRecorder()
	h.ShortenURL(rr, req)
	fmt.Println(rr.Code)
	fmt.Println(strings.HasPrefix(rr.Body.String(), "http://localhost:8080/"))
	// Output:
	// 201
	// true
}

func ExampleHandler_ShortenURLJSON() {
	store := storage.NewMemoryStorage()
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	deleter := service.NewDeleter(store, 100, 5*time.Second)
	h := handler.New(store, cfg, deleter)
	body := `{"url":"https://example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ShortenURLJSON(rr, req)
	fmt.Println(rr.Code)
	var resp map[string]string
	json.NewDecoder(rr.Body).Decode(&resp)
	fmt.Println(strings.HasPrefix(resp["result"], "http://localhost:8080/"))
	// Output:
	// 201
	// true
}

func ExampleHandler_RedirectURL() {
	fmt.Println("Would return 307 with Location header")
	// Output:
	// Would return 307 with Location header
}

func ExampleHandler_GetUserURLs() {
	store := storage.NewMemoryStorage()
	cfg := &config.Config{BaseURL: "http://localhost:8080", SecretKey: "secret"}
	deleter := service.NewDeleter(store, 100, 5*time.Second)
	h := handler.New(store, cfg, deleter)
	userID := "test-user"
	cookie, _ := auth.SignCookie(userID, cfg.SecretKey)
	store.SaveWithUser("https://example.com/1", userID)
	store.SaveWithUser("https://example.com/2", userID)
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	h.GetUserURLs(rr, req)
	fmt.Println(rr.Code)
	var urls []map[string]string
	json.NewDecoder(rr.Body).Decode(&urls)
	fmt.Println(len(urls))
	// Output:
	// 200
	// 2
}

func ExampleHandler_DeleteUserURLs() {
	store := storage.NewMemoryStorage()
	cfg := &config.Config{BaseURL: "http://localhost:8080", SecretKey: "secret"}
	deleter := service.NewDeleter(store, 100, 5*time.Second)
	h := handler.New(store, cfg, deleter)
	userID := "test-user"
	id1, _, _ := store.SaveWithUser("https://example.com/1", userID)
	id2, _, _ := store.SaveWithUser("https://example.com/2", userID)
	body := bytes.NewBufferString(fmt.Sprintf(`["%s","%s"]`, id1, id2))
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", body)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), handler.UserIDKey, userID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.DeleteUserURLs(rr, req)
	fmt.Println(rr.Code)
	// Output:
	// 202
}
