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

// Хранилище URL и счетчик с защитой мьютексом
var (
	urlStore   = make(map[string]string)
	urlStoreMu sync.RWMutex
	idCounter  int
)

// generateShortID создает короткий уникальный идентификатор (например, 8 символов)
func generateShortID() string {
	// Генерируем 6 случайных байт и кодируем в base64 URL-вариант (без паддинга)
	// Это даст строку из 8 символов (base64 использует 64 символа, что похоже на требуемый вид)
	b := make([]byte, 6)
	_, err := rand.Read(b)
	if err != nil {
		// В случае ошибки (маловероятно) возвращаем просто инкрементный ID как строку
		idCounter++
		return fmt.Sprintf("%d", idCounter)
	}
	// URLEncoding без паддинга, чтобы не было символа '='
	return base64.URLEncoding.EncodeToString(b)
}

func main() {
	fmt.Println("Starting shortener server on :8080")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// POST /
		if r.Method == http.MethodPost && r.URL.Path == "/" {
			// Читаем тело запроса
			body, err := io.ReadAll(r.Body)
			if err != nil || len(body) == 0 {
				http.Error(w, "Bad request: empty body", http.StatusBadRequest)
				return
			}
			originalURL := string(body)

			// Генерируем короткий идентификатор
			shortID := generateShortID()

			// Сохраняем в хранилище (с защитой от гонок)
			urlStoreMu.Lock()
			urlStore[shortID] = originalURL
			urlStoreMu.Unlock()

			// Формируем сокращенный URL
			shortURL := fmt.Sprintf("http://localhost:8080/%s", shortID)

			// Устанавливаем заголовки и отправляем ответ
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(shortURL))
			return
		}

		// GET /{id}
		if r.Method == http.MethodGet && len(r.URL.Path) > 1 {
			shortID := strings.TrimPrefix(r.URL.Path, "/")

			// Ищем оригинальный URL
			urlStoreMu.RLock()
			originalURL, exists := urlStore[shortID]
			urlStoreMu.RUnlock()

			if !exists {
				http.Error(w, "Bad request: ID not found", http.StatusBadRequest)
				return
			}

			// Возвращаем редирект
			w.Header().Set("Location", originalURL)
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}

		// Все остальные случаи — 400
		http.Error(w, "Bad request", http.StatusBadRequest)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
