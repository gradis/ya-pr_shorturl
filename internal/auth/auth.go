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

const CookieName = "user_id"

var ErrInvalidToken = errors.New("invalid authentication token")

type userIDContextKey struct{}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(string)
	return userID, ok && userID != ""
}

func NewUserID() (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randomBytes), nil
}

func Sign(userID string, secret []byte) (string, error) {
	if userID == "" || len(secret) == 0 {
		return "", ErrInvalidToken
	}

	signature := tokenSignature(userID, secret)
	return userID + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

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
