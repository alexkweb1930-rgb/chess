package platform

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// NewID генерирует короткий идентификатор для MVP.
func NewID() string {
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}

	return hex.EncodeToString(randomBytes)
}
