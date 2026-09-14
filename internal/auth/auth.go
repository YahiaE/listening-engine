package auth

import (
    "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func GenerateToken() string {
    return rand.Text()
}

func EncryptToken(token string) string {
	hashed := sha256.New()
	hashed.Write([]byte(token))
	encrypted := hashed.Sum(nil)
	return hex.EncodeToString(encrypted)
}
