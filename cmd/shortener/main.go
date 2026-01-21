package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// Простое хранилище в памяти
var urlStore = make(map[string]string)
var idCounter = 0

func main() {
	fmt.Println("Starting shortener server on :8080")

	// Обработчик корневого пути
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// POST запрос для сокращения URL
		if r.Method == http.MethodPost && r.URL.Path == "/" {
			body, err := io.ReadAll(r.Body)
			if err != nil || len(body) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			originalURL := string(body)
			idCounter++
			shortID := fmt.Sprintf("%d", idCounter)
			urlStore[shortID] = originalURL

			// Возвращаем сокращенный URL
			shortURL := fmt.Sprintf("http://localhost:8080/%s", shortID)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(shortURL))
			return
		}

		// GET запрос для получения оригинального URL по ID
		if r.Method == http.MethodGet && len(r.URL.Path) > 1 {
			shortID := strings.TrimPrefix(r.URL.Path, "/")
			if originalURL, exists := urlStore[shortID]; exists {
				w.Header().Set("Location", originalURL)
				w.WriteHeader(http.StatusTemporaryRedirect)
				return
			}
			// Если ID не найден - 400
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Все остальные запросы - 400
		w.WriteHeader(http.StatusBadRequest)
	})

	// Запускаем сервер
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
