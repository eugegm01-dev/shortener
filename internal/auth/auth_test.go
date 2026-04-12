package auth

import (
	"net/http/httptest"
	"testing"
)

func TestSignAndVerifyCookie(t *testing.T) {
	secret := "test-secret"
	userID := "user123"

	cookie, err := SignCookie(userID, secret)
	if err != nil {
		t.Fatalf("SignCookie failed: %v", err)
	}
	if cookie.Name != "auth_token" {
		t.Errorf("Expected cookie name 'auth_token', got '%s'", cookie.Name)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	gotUserID, err := VerifyCookie(req, secret)
	if err != nil {
		t.Fatalf("VerifyCookie failed: %v", err)
	}
	if gotUserID != userID {
		t.Errorf("Expected userID '%s', got '%s'", userID, gotUserID)
	}
}

func TestSignCookieEmptySecret(t *testing.T) {
	_, err := SignCookie("user", "")
	if err == nil {
		t.Error("Expected error for empty secret")
	}
}

func TestVerifyCookieInvalidSignature(t *testing.T) {
	secret := "test-secret"
	cookie, _ := SignCookie("user", secret)
	cookie.Value = "invalid.signature"

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(cookie)

	_, err := VerifyCookie(req, secret)
	if err == nil {
		t.Error("Expected error for invalid signature")
	}
}
