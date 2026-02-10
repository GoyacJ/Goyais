package workflow

import (
	"crypto/rand"
	"encoding/hex"
)

func newID(prefix string) string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return prefix
	}
	return prefix + "_" + hex.EncodeToString(buf)
}
