package storage

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/eugegm01-dev/shortener/internal/models"
)

// Storage определяет интерфейс хранилища URL
type Storage interface {
	Save(url string) (string, error)
	Get(id string) (string, error)
	GetAll() ([]models.URL, error)
	Ping() error
	Close() error
}

// MemoryStorage - хранилище в памяти
type MemoryStorage struct {
	mu    sync.RWMutex
	store map[string]string
}

// NewMemoryStorage создает новое хранилище в памяти
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		store: make(map[string]string),
	}
}

// Save сохраняет URL и возвращает его ID
func (s *MemoryStorage) Save(url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("url cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Генерируем ID
	id := generateShortID()

	// Проверяем на коллизии (в реальном приложении нужна более надежная логика)
	if _, exists := s.store[id]; exists {
		// Если ID уже существует, генерируем новый
		id = generateShortID()
	}

	s.store[id] = url
	return id, nil
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

// generateShortID генерирует короткий идентификатор
func generateShortID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", b)
	}
	return base64.URLEncoding.EncodeToString(b)[:8]
}
