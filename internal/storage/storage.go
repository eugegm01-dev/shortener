package storage

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"github.com/eugegm01-dev/shortener/internal/models"
)

// общие ошибки для всего пакета
var (
	errEmptyURL = fmt.Errorf("url cannot be empty")
	errNotFound = fmt.Errorf("url not found")
	errGone     = fmt.Errorf("url has been deleted")
)

// ErrGone экспортируемая ошибка для хендлеров
var ErrGone = errGone

// bufPool для генерации случайных байт
var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 6)
	},
}

// GenerateShortID генерирует случайный короткий идентификатор.
func GenerateShortID() string {
	b := bufPool.Get().([]byte)
	defer bufPool.Put(b)

	if _, err := rand.Read(b); err != nil {
		// fallback
		return base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%x", b)))[:8]
	}

	var sb strings.Builder
	sb.Grow(8)
	enc := base64.NewEncoder(base64.URLEncoding, &sb)
	enc.Write(b)
	enc.Close()
	return sb.String()[:8]
}

// Storage определяет интерфейс хранилища URL
type Storage interface {
	Save(url string) (id string, created bool, err error)
	SaveWithUser(url, userID string) (id string, created bool, err error)
	Get(id string) (string, error)
	GetByUser(userID string) ([]models.UserURL, error)
	GetAll() ([]models.URL, error)
	Ping() error
	Close() error
	SaveBatch(urls []string) ([]string, error)
	DeleteUserURLs(userID string, shortIDs []string) error
}

// MemoryStorage - хранилище в памяти
type MemoryStorage struct {
	mu       sync.RWMutex
	store    map[string]string          // id -> original_url
	urlToID  map[string]string          // original_url -> id
	userURLs map[string]map[string]bool // user_id -> set of url_ids
	deleted  map[string]bool            // id -> deleted flag
}

// NewMemoryStorage создает новое хранилище в памяти
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		store:    make(map[string]string),
		urlToID:  make(map[string]string),
		userURLs: make(map[string]map[string]bool),
		deleted:  make(map[string]bool),
	}
}

// Save сохраняет URL и возвращает его ID
func (s *MemoryStorage) Save(url string) (string, bool, error) {
	return s.SaveWithUser(url, "") // Без пользователя
}

// SaveWithUser сохраняет URL с привязкой к userID
func (s *MemoryStorage) SaveWithUser(url, userID string) (string, bool, error) {
	if url == "" {
		return "", false, errEmptyURL
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Если уже есть – возвращаем существующий id
	if id, ok := s.urlToID[url]; ok {
		// Привязываем к пользователю, если он есть
		if userID != "" {
			if s.userURLs[userID] == nil {
				s.userURLs[userID] = make(map[string]bool)
			}
			s.userURLs[userID][id] = true
		}
		return id, false, nil
	}

	// Генерируем новый ID
	id := GenerateShortID()
	for {
		if _, exists := s.store[id]; !exists {
			break
		}
		id = GenerateShortID()
	}

	// Сохраняем URL
	s.store[id] = url
	s.urlToID[url] = id

	// Привязываем к пользователю, если он есть
	if userID != "" {
		if s.userURLs[userID] == nil {
			s.userURLs[userID] = make(map[string]bool)
		}
		s.userURLs[userID][id] = true
	}

	return id, true, nil
}

// Get возвращает URL по ID
func (s *MemoryStorage) Get(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.deleted[id] {
		return "", errGone
	}
	url, exists := s.store[id]
	if !exists {
		return "", errNotFound
	}
	return url, nil
}

// GetByUser возвращает все ссылки пользователя
func (s *MemoryStorage) GetByUser(userID string) ([]models.UserURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	urlSet, exists := s.userURLs[userID]
	if !exists {
		return []models.UserURL{}, nil
	}
	urls := make([]models.UserURL, 0)
	for id := range urlSet {
		if s.deleted[id] {
			continue
		}
		if originalURL, ok := s.store[id]; ok {
			urls = append(urls, models.UserURL{
				ShortURL:    "http://localhost:8080/" + id,
				OriginalURL: originalURL,
			})
		}
	}
	return urls, nil
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

// SaveBatch сохраняет несколько URL
func (s *MemoryStorage) SaveBatch(urls []string) ([]string, error) {
	return s.SaveBatchWithUser(urls, "") // Без пользователя
}

// SaveBatchWithUser сохраняет несколько URL с привязкой к пользователю
func (s *MemoryStorage) SaveBatchWithUser(urls []string, userID string) ([]string, error) {
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
			// Привязываем к пользователю
			if userID != "" {
				if s.userURLs[userID] == nil {
					s.userURLs[userID] = make(map[string]bool)
				}
				s.userURLs[userID][id] = true
			}
			continue
		}

		// Генерируем новый ID
		id := GenerateShortID()
		for {
			if _, exists := s.store[id]; !exists {
				break
			}
			id = GenerateShortID()
		}

		// Сохраняем
		s.store[id] = url
		s.urlToID[url] = id

		// Привязываем к пользователю
		if userID != "" {
			if s.userURLs[userID] == nil {
				s.userURLs[userID] = make(map[string]bool)
			}
			s.userURLs[userID][id] = true
		}

		ids = append(ids, id)
	}

	return ids, nil
}

// DeleteUserURLs помечает URL пользователя как удалённые
func (s *MemoryStorage) DeleteUserURLs(userID string, shortIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range shortIDs {
		// Проверяем, принадлежит ли ссылка пользователю
		if _, ok := s.userURLs[userID]; ok && s.userURLs[userID][id] {
			s.deleted[id] = true
		}
	}
	return nil
}
