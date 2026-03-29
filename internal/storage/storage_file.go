package storage

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/eugegm01-dev/shortener/internal/models"
)

type fileRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type FileStorage struct {
    mu       sync.RWMutex
    store    map[string]string // short -> original
    urlToID  map[string]string // original -> short
    filePath string
}

func NewFileStorage(filePath string) (*FileStorage, error) {
    fs := &FileStorage{
        store:    make(map[string]string),
        urlToID:  make(map[string]string),
        filePath: filePath,
    }
    if err := fs.load(); err != nil {
        return nil, err
    }
    return fs, nil
}

// load загружает данные из файла (вызывается один раз при инициализации).
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
        fs.store[rec.ShortURL] = rec.OriginalURL
        fs.urlToID[rec.OriginalURL] = rec.ShortURL
    }
    return nil
}
// save записывает текущее состояние store в файл.
// Вызывается только при уже захваченном мьютексе на запись (из Save).
func (fs *FileStorage) save() error {
	// преобразуем store в срез fileRecord
	records := make([]fileRecord, 0, len(fs.store))
	for shortURL, originalURL := range fs.store {
		records = append(records, fileRecord{
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
	}
	// добавляем UUID (порядковый номер) для красивого вывода
	for i := range records {
		records[i].UUID = itoa(i + 1)
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

// itoa — простейшее преобразование int в string (без импорта strconv).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func (fs *FileStorage) Save(url string) (string, bool, error) {
    if url == "" {
        return "", false, errEmptyURL
    }
    fs.mu.Lock()
    defer fs.mu.Unlock()

    if id, ok := fs.urlToID[url]; ok {
        return id, false, nil
    }
    id := generateShortID()
    for {
        if _, exists := fs.store[id]; !exists {
            break
        }
        id = generateShortID()
    }
    fs.store[id] = url
    fs.urlToID[url] = id
    if err := fs.save(); err != nil {
        delete(fs.store, id)
        delete(fs.urlToID, url)
        return "", false, err
    }
    return id, true, nil
}

func (fs *FileStorage) Get(id string) (string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	url, ok := fs.store[id]
	if !ok {
		return "", errNotFound
	}
	return url, nil
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
    if len(urls) == 0 {
        return nil, nil
    }

    fs.mu.Lock()
    defer fs.mu.Unlock()

    ids := make([]string, 0, len(urls))

    for _, url := range urls {
        if url == "" {
            return nil, errEmptyURL
        }

        id := generateShortID()
        for {
            if _, exists := fs.store[id]; !exists {
                break
            }
            id = generateShortID()
        }
        fs.store[id] = url
        ids = append(ids, id)
    }

    // Сохраняем всё одним запросом в файл
    if err := fs.save(); err != nil {
        // Откат: удаляем добавленные записи
        for _, id := range ids {
            delete(fs.store, id)
        }
        return nil, err
    }

    return ids, nil
}