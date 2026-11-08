package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

// CookieName задаёт имя cookie с подписанным идентификатором пользователя.
const CookieName = "user_id"

// ErrInvalidToken означает некорректный токен, подпись или пустые параметры подписи.
var ErrInvalidToken = errors.New("invalid authentication token")

type userIDContextKey struct{}

// WithUserID возвращает дочерний контекст с идентификатором пользователя.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

// UserIDFromContext извлекает непустой ID пользователя и признак его наличия.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(string)
	return userID, ok && userID != ""
}

// NewUserID генерирует криптографически случайный ID из 16 байт в hex-формате.
func NewUserID() (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes), nil
}

// Sign подписывает ID пользователя ключом secret с помощью HMAC-SHA256.
// Пустой ID или ключ приводит к ErrInvalidToken.
func Sign(userID string, secret []byte) (string, error) {
	if userID == "" || len(secret) == 0 {
		return "", ErrInvalidToken
	}

	signature := tokenSignature(userID, secret)
	return userID + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// Verify проверяет формат и подпись токена и возвращает ID пользователя.
// Подписи сравниваются за постоянное время; неверный токен даёт ErrInvalidToken.
func Verify(token string, secret []byte) (string, error) {
	userID, encodedSignature, ok := strings.Cut(token, ".")
	if !ok || userID == "" || encodedSignature == "" || len(secret) == 0 {
		return "", ErrInvalidToken
	}

	signature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return "", ErrInvalidToken
	}

	expectedSignature := tokenSignature(userID, secret)
	if !hmac.Equal(signature, expectedSignature) {
		return "", ErrInvalidToken
	}

	return userID, nil
}

func tokenSignature(userID string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(userID))
	return mac.Sum(nil)
}
