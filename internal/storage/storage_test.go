package storage

import (
	"testing"
)

func TestMemoryStorage(t *testing.T) {
	store := NewMemoryStorage()

	// Test Save
	url := "https://example.com"
id, created, err := store.Save(url)
_ = created

	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if id == "" {
		t.Error("Expected non-empty ID")
	}

	// Test Get
	retrievedURL, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrievedURL != url {
		t.Errorf("Get() = %v, want %v", retrievedURL, url)
	}

	// Test Get non-existent
	_, err = store.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent ID")
	}

	// Test GetAll
	urls, err := store.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(urls) != 1 {
		t.Errorf("GetAll() returned %v items, want 1", len(urls))
	}

	// Test Ping
	if err := store.Ping(); err != nil {
		t.Errorf("Ping() failed: %v", err)
	}
}

func TestMemoryStorageEmptyURL(t *testing.T) {
    store := NewMemoryStorage()
    _, _, err := store.Save("")
    if err == nil {
        t.Error("Expected error for empty URL")
    }
}