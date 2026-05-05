package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestSubjectNotify(t *testing.T) {
	s := NewSubject()
	mock := &mockObserver{ch: make(chan Event, 1)}
	s.Attach(mock)

	event := Event{TS: 123, Action: "test", UserID: "u1", URL: "http://example.com"}
	s.Notify(event)

	select {
	case got := <-mock.ch:
		if got.TS != event.TS || got.Action != event.Action {
			t.Errorf("Event mismatch: %+v", got)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("No event received within timeout")
	}
}

type mockObserver struct {
	ch chan Event
}

func (m *mockObserver) Send(e Event) error {
	m.ch <- e
	return nil
}

func TestFileObserver(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/audit.log"

	fo, err := NewFileObserver(path)
	if err != nil {
		t.Fatal(err)
	}
	event := Event{TS: 100, Action: "shorten", UserID: "u1", URL: "http://x.com"}
	if err := fo.Send(event); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got Event
	if err := json.Unmarshal(data[:len(data)-1], &got); err != nil {
		t.Fatal(err)
	}
	if got.TS != 100 {
		t.Errorf("Expected TS 100, got %d", got.TS)
	}
}

func TestHTTPObserver(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var e Event
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if e.Action != "test" {
			http.Error(w, "bad action", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ho := NewHTTPObserver(ts.URL)
	err := ho.Send(Event{TS: 1, Action: "test"})
	if err != nil {
		t.Errorf("Send failed: %v", err)
	}
}
