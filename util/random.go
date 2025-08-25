package util

import (
	"crypto/rand"
	"encoding/base64"
)

func RandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Converti in stringa base64 (sarà più lunga del length originale)
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
