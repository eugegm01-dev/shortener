package storage

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/eugegm01-dev/shortener/internal/models"
)

// общие ошибки для всего пакета
var (
    errEmptyURL = fmt.Errorf("url cannot be empty")
    errNotFound = fmt.Errorf("url not found")
)

// generateShortID – общая функция для всех хранилищ
func generateShortID() string {
    b := make([]byte, 6)
    if _, err := rand.Read(b); err != nil {
        return fmt.Sprintf("%x", b)
    }
    return base64.URLEncoding.EncodeToString(b)[:8]
}


// Storage определяет интерфейс хранилища URL
type Storage interface {
    Save(url string) (id string, created bool, err error)
    Get(id string) (string, error)
    GetAll() ([]models.URL, error)
    Ping() error
    Close() error
    // Новый метод для батч-операций
    SaveBatch(urls []string) ([]string, error) // возвращает список ID в том же порядке
}
// MemoryStorage - хранилище в памяти
type MemoryStorage struct {
    mu      sync.RWMutex
    store   map[string]string // id -> original_url
    urlToID map[string]string // original_url -> id
}


// NewMemoryStorage создает новое хранилище в памяти
func NewMemoryStorage() *MemoryStorage {
    return &MemoryStorage{
        store:   make(map[string]string),
        urlToID: make(map[string]string),
    }
}

// Save сохраняет URL и возвращает его ID
func (s *MemoryStorage) Save(url string) (string, bool, error) {
    if url == "" {
        return "", false, errEmptyURL
    }
    s.mu.Lock()
    defer s.mu.Unlock()
    // Если уже есть – возвращаем существующий id
    if id, ok := s.urlToID[url]; ok {
        return id, false, nil
    }
    id := generateShortID()
    for {
        if _, exists := s.store[id]; !exists {
            break
        }
        id = generateShortID()
    }
    s.store[id] = url
    s.urlToID[url] = id
    return id, true, nil
}

// Get возвращает URL по ID
func (s *MemoryStorage) Get(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, exists := s.store[id]
	if !exists {
		return "", fmt.Errorf("url not found")
	}
	return url, nil
}

// GetAll возвращает все сохраненные URL
func (s *MemoryStorage) GetAll() ([]models.URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	urls := make([]models.URL, 0, len(s.store))
	for id, url := range s.store {
		urls = append(urls, models.URL{
			ID:  id,
			URL: url,
		})
	}
	return urls, nil
}

// Ping проверяет доступность хранилища
func (s *MemoryStorage) Ping() error {
	return nil
}

// Close освобождает ресурсы
func (s *MemoryStorage) Close() error {
	return nil
}


func (s *MemoryStorage) SaveBatch(urls []string) ([]string, error) {
    if len(urls) == 0 {
        return nil, nil
    }
    s.mu.Lock()
    defer s.mu.Unlock()

    ids := make([]string, 0, len(urls))
    for _, url := range urls {
        if url == "" {
            return nil, errEmptyURL
        }
        // Если уже есть – используем существующий id
        if id, ok := s.urlToID[url]; ok {
            ids = append(ids, id)
            continue
        }
        id := generateShortID()
        for {
            if _, exists := s.store[id]; !exists {
                break
            }
            id = generateShortID()
        }
        s.store[id] = url
        s.urlToID[url] = id
        ids = append(ids, id)
    }
    return ids, nil
}
