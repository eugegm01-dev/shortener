package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/eugegm01-dev/shortener/internal/models"
)

// fileRecord represents a single record in the JSON file.
type fileRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// FileStorage implements Storage with persistence to a JSON file.
type FileStorage struct {
	mu       sync.RWMutex
	store    map[string]string // short URL -> original URL
	filePath string
}

// NewFileStorage creates a new FileStorage and loads existing data from the file.
func NewFileStorage(filePath string) (*FileStorage, error) {
	fs := &FileStorage{
		store:    make(map[string]string),
		filePath: filePath,
	}
	if err := fs.load(); err != nil {
		return nil, err
	}
	return fs, nil
}

// load reads the JSON file and populates the store.
func (fs *FileStorage) load() error {
	file, err := os.Open(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // file doesn't exist, start empty
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
	}
	return nil
}

// save writes the current store to the JSON file.
func (fs *FileStorage) save() error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	// Convert store to slice of fileRecord
	records := make([]fileRecord, 0, len(fs.store))
	i := 1
	for shortURL, originalURL := range fs.store {
		records = append(records, fileRecord{
			UUID:        "", // we don't store UUID, assign sequential for output
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
		i++
	}
	// Assign sequential UUIDs (1,2,3...)
	for j := range records {
		records[j].UUID = itoa(j + 1) // simple conversion, but we need a function
	}

	// Write to file (create or truncate)
	file, err := os.Create(fs.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // pretty print like example
	return encoder.Encode(records)
}

// Simple int to string converter (instead of strconv.Itoa to avoid extra import)
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

// Save stores a URL and returns its short ID. Writes to file.
func (fs *FileStorage) Save(url string) (string, error) {
	if url == "" {
		return "", errEmptyURL
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Generate a unique ID
	id := generateShortID()
	for {
		if _, exists := fs.store[id]; !exists {
			break
		}
		id = generateShortID() // collision, try again
	}
	fs.store[id] = url

	// Persist to file
	if err := fs.save(); err != nil {
		// If save fails, we should probably rollback? For simplicity, we just return error.
		// In a real app, you might want to remove the entry from memory.
		delete(fs.store, id)
		return "", err
	}

	return id, nil
}

// Get returns the original URL for a given short ID.
func (fs *FileStorage) Get(id string) (string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	url, ok := fs.store[id]
	if !ok {
		return "", errNotFound
	}
	return url, nil
}

// GetAll returns all stored URLs.
func (fs *FileStorage) GetAll() ([]models.URL, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	urls := make([]models.URL, 0, len(fs.store))
	for id, url := range fs.store {
		urls = append(urls, models.URL{
			ID:  id,
			URL: url,
		})
	}
	return urls, nil
}

// Ping always returns nil (file storage is considered always available).
func (fs *FileStorage) Ping() error {
	return nil
}

// Close performs any necessary cleanup. For file storage, nothing is needed.
func (fs *FileStorage) Close() error {
	return nil
}

// Define errors used in storage (could be placed in a common place)
var (
	errEmptyURL = fmt.Errorf("url cannot be empty")
	errNotFound = fmt.Errorf("url not found")
)
