package utils

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/eichiarakaki/aegis/internals/logger"
	"github.com/google/uuid"
)

func GenerateUUID() (string, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		logger.Errorf("Failed to generate UUID: %v", err)
		return "", err
	}
	return id.String(), nil
}

func GenerateSHA1(data string) string {
	h := sha1.New()
	h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func GetShortHash(fullHash string) string {
	return fullHash[:7]
}

func GenerateSecureToken() string {
	length := 16 // 16 bytes = 128 bits
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
