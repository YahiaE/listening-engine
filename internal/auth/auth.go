package auth

import (
    "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"fmt"
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

func GenerateOTP(length int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(length)), nil)
	
	num, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	// leading zeros to match the requested length
	return fmt.Sprintf("%0*d", length, num), nil
}
