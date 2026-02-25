package httpapi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"strings"
)

// Keep this seed stable across releases so existing encrypted secrets stay decryptable.
const defaultSecretSeed = "goyais-v0.4.0-resource-secret"

func encryptSecret(plain string) (string, error) {
	if strings.TrimSpace(plain) == "" {
		return "", nil
	}
	key := resolveSecretKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	buf := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(buf), nil
}

func decryptSecret(encoded string) (string, error) {
	if strings.TrimSpace(encoded) == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	key := resolveSecretKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext is too short")
	}
	nonce := raw[:gcm.NonceSize()]
	ciphertext := raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func maskSecret(secret string) string {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 6 {
		return "***"
	}
	return trimmed[:3] + "..." + trimmed[len(trimmed)-2:]
}

func resolveSecretKey() []byte {
	seed := strings.TrimSpace(os.Getenv("RESOURCE_SECRET_KEY"))
	if seed == "" {
		seed = defaultSecretSeed
	}
	sum := sha256.Sum256([]byte(seed))
	return sum[:]
}
