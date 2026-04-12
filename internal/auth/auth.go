package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

const cookieName = "auth_token"

// SignCookie создаёт подписанную куку: base64(userID) + "." + base64(HMAC)
func SignCookie(userID, secretKey string) (*http.Cookie, error) {
	if secretKey == "" {
		return nil, errors.New("secret key is empty")
	}

	// Кодируем user_id в base64
	encodedID := base64.URLEncoding.EncodeToString([]byte(userID))

	// Считаем HMAC-SHA256 от encodedID
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(encodedID))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// Формируем значение куки: encodedID.signature
	value := encodedID + "." + signature

	return &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // true в продакшене с HTTPS
		SameSite: http.SameSiteLaxMode,
	}, nil
}

// VerifyCookie проверяет подпись и возвращает user_id
func VerifyCookie(r *http.Request, secretKey string) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", err // Просто возвращаем ошибку, если куки нет
	}

	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid cookie format")
	}

	encodedID, signature := parts[0], parts[1]

	// Проверяем HMAC
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(encodedID))
	expectedSig := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return "", errors.New("invalid cookie signature")
	}

	// Декодируем user_id
	userID, err := base64.URLEncoding.DecodeString(encodedID)
	if err != nil {
		return "", errors.New("failed to decode user ID")
	}

	return string(userID), nil
}
