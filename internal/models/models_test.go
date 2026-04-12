package models

import (
	"encoding/json"
	"testing"
)

func TestStructuresSerialization(t *testing.T) {
	req := ShortenRequest{URL: "https://a.com"}
	data, _ := json.Marshal(req)
	if string(data) != `{"url":"https://a.com"}` {
		t.Errorf("Unexpected JSON: %s", data)
	}

	resp := ShortenResponse{Result: "http://localhost/abc"}
	data, _ = json.Marshal(resp)
	if string(data) != `{"result":"http://localhost/abc"}` {
		t.Errorf("Unexpected JSON: %s", data)
	}
}
