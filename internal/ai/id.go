package ai

import (
	"crypto/rand"
	"encoding/hex"
)

func newID(prefix string) string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return prefix + "_" + hex.EncodeToString(buf[:])
}
