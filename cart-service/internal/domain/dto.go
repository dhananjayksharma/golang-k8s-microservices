package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/google/uuid"
)

func UUIDToBin16(u uuid.UUID) []byte {
	b := make([]byte, 16)
	copy(b, u[:])
	return b
}

func Bin16ToUUID(b []byte) (uuid.UUID, error) {
	return uuid.FromBytes(b)
}

func HashRequest(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}
