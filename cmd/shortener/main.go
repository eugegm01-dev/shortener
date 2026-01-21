package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Хранилище URL с защитой мьютексом
type URLStore struct {
	store map[string]string
	mu    sync.RWMutex
}

func NewURLStore() *URLStore {
	return &URLStore{
		store: make(map[string]string),
	}
}

func (s *URLStore) Save(url string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Генерация случайного короткого ID (8 символов)
	shortID := generateShortID()
	s.store[shortID] = url
	return shortID
}

func (s *URLStore) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, exists := s.store[shortID]
	return url, exists
}

// Генерация случайного короткого ID (например, EwHXdJfB)
func generateShortID() string {
	// Генерируем 6 случайных байт для 8 символов в base64
	b := make([]byte, 6)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback: используем временную метку или другой метод
		return fmt.Sprintf("%x", b)
	}
	// URLEncoding без паддинга, чтобы избежать символа '='
	return base64.URLEncoding.EncodeToString(b)[:8]
}

func main() {
	fmt.Println("Starting shortener server on :8080")

	urlStore := NewURLStore()

	// Обработчик корневого пути
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s request to %s", r.Method, r.URL.Path)

		// POST запрос для сокращения URL
		if r.Method == http.MethodPost && r.URL.Path == "/" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				log.Printf("Error reading body: %v", err)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}

			originalURL := strings.TrimSpace(string(body))
			if originalURL == "" {
				log.Printf("Empty URL received")
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}

			log.Printf("Shortening URL: %s", originalURL)
			shortID := urlStore.Save(originalURL)
			shortURL := fmt.Sprintf("http://localhost:8080/%s", shortID)

			log.Printf("Created short URL: %s -> %s", shortID, originalURL)

			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(shortURL))
			return
		}

		// GET запрос для получения оригинального URL по ID
		if r.Method == http.MethodGet && len(r.URL.Path) > 1 {
			shortID := strings.TrimPrefix(r.URL.Path, "/")

			log.Printf("Looking up ID: %s", shortID)
			originalURL, exists := urlStore.Get(shortID)

			if !exists {
				log.Printf("ID not found: %s", shortID)
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}

			log.Printf("Redirecting %s -> %s", shortID, originalURL)
			w.Header().Set("Location", originalURL)
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}

		// Все остальные запросы - 400
		log.Printf("Invalid request: %s %s", r.Method, r.URL.Path)
		http.Error(w, "Bad Request", http.StatusBadRequest)
	})

	log.Println("Server is ready. Listening on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
