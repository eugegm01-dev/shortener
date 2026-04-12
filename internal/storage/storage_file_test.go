package storage

import (
	"testing"
)

func TestFileStorage_SaveAndGet(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/storage.json"

	fs, err := NewFileStorage(path)
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Close()

	url := "https://example.com"
	id, created, err := fs.Save(url)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Error("Expected created=true for new URL")
	}

	got, err := fs.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got != url {
		t.Errorf("Expected %s, got %s", url, got)
	}
}

func TestFileStorage_SaveDuplicate(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/storage.json"

	fs, _ := NewFileStorage(path)
	defer fs.Close()

	url := "https://dup.com"
	id1, _, _ := fs.Save(url)
	id2, created2, _ := fs.Save(url)

	if id1 != id2 {
		t.Error("IDs should be same for duplicate URL")
	}
	if created2 {
		t.Error("Expected created=false for duplicate")
	}
}

func TestFileStorage_GetByUser(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/storage.json"

	fs, _ := NewFileStorage(path)
	defer fs.Close()

	user1 := "user1"
	fs.SaveWithUser("https://u1.com/1", user1)
	fs.SaveWithUser("https://u1.com/2", user1)
	fs.SaveWithUser("https://u2.com/1", "user2")

	urls, err := fs.GetByUser(user1)
	if err != nil {
		t.Fatal(err)
	}
	if len(urls) != 2 {
		t.Errorf("Expected 2 URLs, got %d", len(urls))
	}
}

func TestFileStorage_DeleteUserURLs(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/storage.json"

	fs, _ := NewFileStorage(path)
	defer fs.Close()

	user := "user1"
	id1, _, _ := fs.SaveWithUser("https://del.com/1", user)
	id2, _, _ := fs.SaveWithUser("https://del.com/2", user)

	err := fs.DeleteUserURLs(user, []string{id1})
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.Get(id1)
	if err != errGone {
		t.Errorf("Expected errGone, got %v", err)
	}
	_, err = fs.Get(id2)
	if err != nil {
		t.Errorf("Non-deleted URL should be accessible: %v", err)
	}
}

func TestFileStorage_Persistence(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/storage.json"

	fs1, _ := NewFileStorage(path)
	url := "https://persist.com"
	id, _, _ := fs1.Save(url)
	fs1.Close()

	fs2, _ := NewFileStorage(path)
	defer fs2.Close()

	got, err := fs2.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got != url {
		t.Errorf("Expected %s, got %s", url, got)
	}
}
