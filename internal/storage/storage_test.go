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

func TestMemoryStorage_GetByUser(t *testing.T) {
	store := NewMemoryStorage()
	userID := "user1"

	// Сохраняем URL от пользователя
	id1, _, _ := store.SaveWithUser("https://example.com/1", userID)
	id2, _, _ := store.SaveWithUser("https://example.com/2", userID)

	urls, err := store.GetByUser(userID)
	if err != nil {
		t.Fatalf("GetByUser failed: %v", err)
	}
	if len(urls) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(urls))
	}

	// Проверяем содержимое (без учёта порядка)
	found := make(map[string]bool)
	for _, u := range urls {
		found[u.ShortURL] = true
	}
	if !found["http://localhost:8080/"+id1] || !found["http://localhost:8080/"+id2] {
		t.Error("Missing expected short URLs")
	}
}

func TestMemoryStorage_DeleteUserURLs(t *testing.T) {
	store := NewMemoryStorage()
	userID := "user1"

	id1, _, _ := store.SaveWithUser("https://example.com/1", userID)
	id2, _, _ := store.SaveWithUser("https://example.com/2", userID)

	err := store.DeleteUserURLs(userID, []string{id1})
	if err != nil {
		t.Fatalf("DeleteUserURLs failed: %v", err)
	}

	_, err = store.Get(id1)
	if err != errGone {
		t.Errorf("Expected errGone for deleted URL, got %v", err)
	}

	_, err = store.Get(id2)
	if err != nil {
		t.Errorf("Non-deleted URL should be accessible, got error: %v", err)
	}
}

func TestMemoryStorage_SaveBatch(t *testing.T) {
	store := NewMemoryStorage()
	urls := []string{"https://a.com", "https://b.com"}
	ids, err := store.SaveBatch(urls)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Errorf("Expected 2 ids, got %d", len(ids))
	}
	// Проверим, что URLs сохранились
	for _, id := range ids {
		_, err := store.Get(id)
		if err != nil {
			t.Errorf("Get failed for %s: %v", id, err)
		}
	}
}

func TestMemoryStorage_PingAndClose(t *testing.T) {
	store := NewMemoryStorage()
	if err := store.Ping(); err != nil {
		t.Error("Ping should not fail")
	}
	if err := store.Close(); err != nil {
		t.Error("Close should not fail")
	}
}

func TestMemoryStorage_GetAll(t *testing.T) {
	store := NewMemoryStorage()
	store.Save("https://1.com")
	store.Save("https://2.com")
	all, err := store.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(all))
	}
}
func TestMemoryStorage_SaveBatchWithUser(t *testing.T) {
	store := NewMemoryStorage()
	user := "user1"
	urls := []string{"https://a.com", "https://b.com"}
	ids, err := store.SaveBatchWithUser(urls, user)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 {
		t.Errorf("Expected 2 ids, got %d", len(ids))
	}
	userURLs, _ := store.GetByUser(user)
	if len(userURLs) != 2 {
		t.Errorf("Expected 2 user URLs, got %d", len(userURLs))
	}
}

func TestMemoryStorage_GetByUser_NoURLs(t *testing.T) {
	store := NewMemoryStorage()
	urls, err := store.GetByUser("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 0 {
		t.Errorf("Expected empty slice, got %v", urls)
	}
}

func TestMemoryStorage_GenerateShortID_Length(t *testing.T) {
	id := GenerateShortID()
	if len(id) != 8 {
		t.Errorf("Expected ID length 8, got %d", len(id))
	}
}
