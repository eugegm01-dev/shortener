package storage

import (
	"encoding/json"
	"os"
	"strconv"
	"sync"

	"github.com/eugegm01-dev/shortener/internal/models"
)

type fileRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
	IsDeleted   bool   `json:"is_deleted,omitempty"`
}
type FileStorage struct {
	mu       sync.RWMutex
	store    map[string]string
	urlToID  map[string]string
	userURLs map[string]map[string]bool
	deleted  map[string]bool // ← добавьте
	filePath string
}

// NewFileStorage creates a file‑based storage that persists data to the given JSON file.
func NewFileStorage(filePath string) (*FileStorage, error) {
	fs := &FileStorage{
		store:    make(map[string]string),
		urlToID:  make(map[string]string),
		userURLs: make(map[string]map[string]bool),
		deleted:  make(map[string]bool),
		filePath: filePath,
	}
	if err := fs.load(); err != nil {
		return nil, err
	}
	return fs, nil
}

func (fs *FileStorage) load() error {
	file, err := os.Open(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var records []fileRecord
	if err := json.NewDecoder(file).Decode(&records); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, rec := range records {
		// Извлекаем ID из ShortURL (формат: "http://localhost:8080/{id}")
		shortURL := rec.ShortURL
		id := shortURL
		if len(shortURL) > 20 { // Пример: "http://localhost:8080/abc123"
			id = shortURL[len("http://localhost:8080/"):]
		}

		fs.store[id] = rec.OriginalURL
		fs.urlToID[rec.OriginalURL] = id

		// Восстанавливаем привязки к пользователю
		if rec.UserID != "" {
			if fs.userURLs[rec.UserID] == nil {
				fs.userURLs[rec.UserID] = make(map[string]bool)
			}
			fs.userURLs[rec.UserID][id] = true
		}
	}
	return nil
}

func (fs *FileStorage) save() error {
	records := make([]fileRecord, 0, len(fs.store))
	for id, originalURL := range fs.store {
		// Находим user_id для этой ссылки
		var userID string
		for uID, urlSet := range fs.userURLs {
			if urlSet[id] {
				userID = uID
				break
			}
		}

		records = append(records, fileRecord{
			UUID:        strconv.Itoa(len(records) + 1), // ← используем strconv
			ShortURL:    "http://localhost:8080/" + id,
			OriginalURL: originalURL,
			UserID:      userID,
			IsDeleted:   fs.deleted[id],
		})
	}

	file, err := os.Create(fs.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}

func (fs *FileStorage) Save(url string) (string, bool, error) {
	return fs.SaveWithUser(url, "")
}

func (fs *FileStorage) SaveWithUser(url, userID string) (string, bool, error) {
	if url == "" {
		return "", false, errEmptyURL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Проверяем, существует ли уже URL
	if id, ok := fs.urlToID[url]; ok {
		// Привязываем существующую ссылку к пользователю
		if userID != "" {
			if fs.userURLs[userID] == nil {
				fs.userURLs[userID] = make(map[string]bool)
			}
			fs.userURLs[userID][id] = true
			// Сохраняем изменения в файл
			if err := fs.save(); err != nil {
				return "", false, err
			}
		}
		return id, false, nil
	}

	// Генерируем новый ID
	id := GenerateShortID()
	for {
		if _, exists := fs.store[id]; !exists {
			break
		}
		id = GenerateShortID()
	}

	// Сохраняем новую запись
	fs.store[id] = url
	fs.urlToID[url] = id

	// Привязываем к пользователю, если он есть
	if userID != "" {
		if fs.userURLs[userID] == nil {
			fs.userURLs[userID] = make(map[string]bool)
		}
		fs.userURLs[userID][id] = true
	}

	// Сохраняем в файл
	if err := fs.save(); err != nil {
		// Откатываем изменения в памяти при ошибке записи
		delete(fs.store, id)
		delete(fs.urlToID, url)
		if userID != "" && fs.userURLs[userID] != nil {
			delete(fs.userURLs[userID], id)
		}
		return "", false, err
	}

	return id, true, nil
}

func (fs *FileStorage) Get(id string) (string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	if fs.deleted[id] {
		return "", errGone
	}
	url, ok := fs.store[id]
	if !ok {
		return "", errNotFound
	}
	return url, nil
}

// GetByUser возвращает все ссылки пользователя
func (fs *FileStorage) GetByUser(userID string) ([]models.UserURL, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	urlSet, exists := fs.userURLs[userID]
	if !exists || len(urlSet) == 0 {
		return []models.UserURL{}, nil
	}

	urls := make([]models.UserURL, 0, len(urlSet))
	for id := range urlSet {
		if originalURL, ok := fs.store[id]; ok {
			urls = append(urls, models.UserURL{
				ShortURL:    "http://localhost:8080/" + id, // Базовый URL будет заменен в хендлере
				OriginalURL: originalURL,
			})
		}
	}

	return urls, nil
}

func (fs *FileStorage) GetAll() ([]models.URL, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	urls := make([]models.URL, 0, len(fs.store))
	for id, url := range fs.store {
		urls = append(urls, models.URL{ID: id, URL: url})
	}
	return urls, nil
}

func (fs *FileStorage) Ping() error {
	return nil
}

func (fs *FileStorage) Close() error {
	return nil
}

func (fs *FileStorage) SaveBatch(urls []string) ([]string, error) {
	return fs.SaveBatchWithUser(urls, "")
}

func (fs *FileStorage) SaveBatchWithUser(urls []string, userID string) ([]string, error) {
	if len(urls) == 0 {
		return nil, nil
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	ids := make([]string, 0, len(urls))
	newRecords := make(map[string]string) // Для отката при ошибке

	for _, url := range urls {
		if url == "" {
			return nil, errEmptyURL
		}

		// Проверяем, существует ли уже URL
		if id, ok := fs.urlToID[url]; ok {
			ids = append(ids, id)
			// Привязываем существующую ссылку к пользователю
			if userID != "" {
				if fs.userURLs[userID] == nil {
					fs.userURLs[userID] = make(map[string]bool)
				}
				fs.userURLs[userID][id] = true
			}
			continue
		}

		// Генерируем новый ID
		id := GenerateShortID()
		for {
			if _, exists := fs.store[id]; !exists {
				break
			}
			id = GenerateShortID()
		}

		// Сохраняем во временную структуру
		fs.store[id] = url
		fs.urlToID[url] = id
		newRecords[id] = url

		// Привязываем к пользователю
		if userID != "" {
			if fs.userURLs[userID] == nil {
				fs.userURLs[userID] = make(map[string]bool)
			}
			fs.userURLs[userID][id] = true
		}

		ids = append(ids, id)
	}

	// Сохраняем всё одним запросом в файл
	if err := fs.save(); err != nil {
		// Откат: удаляем добавленные записи
		for id := range newRecords {
			url := newRecords[id]
			delete(fs.store, id)
			delete(fs.urlToID, url)
			if userID != "" && fs.userURLs[userID] != nil {
				delete(fs.userURLs[userID], id)
			}
		}
		return nil, err
	}

	return ids, nil
}

func (fs *FileStorage) DeleteUserURLs(userID string, shortIDs []string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	for _, id := range shortIDs {
		if _, ok := fs.userURLs[userID]; ok && fs.userURLs[userID][id] {
			fs.deleted[id] = true
		}
	}
	return fs.save() // сохраняем изменения
}
