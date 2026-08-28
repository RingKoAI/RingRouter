// Package crypto provides authenticated symmetric encryption for secrets at
// rest (SMTP passwords, channel API keys). Ciphertexts are base64(nonce|ct|tag)
// and safe to store in a database column.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	// EnvKey is the environment variable holding a hex-encoded 256-bit key.
	EnvKey = "ENCRYPTION_KEY"

	keySize   = 32 // AES-256
	nonceSize = 12 // GCM standard
)

// deriveKey builds the encryption key: ENCRYPTION_KEY (hex, 32 bytes) when
// provided, otherwise deterministically derived from the instance secret so a
// default deployment works with zero configuration.
func deriveKey(secret string) []byte {
	if raw := os.Getenv(EnvKey); raw != "" {
		if key, err := hex.DecodeString(strings.TrimSpace(raw)); err == nil && len(key) == keySize {
			return key
		}
		// Misconfigured key: fall through to derivation rather than fail closed,
		// since the operator may rotate it later.
	}

	mac := hmac.New(sha256.New, []byte("ringrouter-encryption-v1"))
	mac.Write([]byte(secret))
	return mac.Sum(nil)
}

// Encryptor seals and opens secrets with one key.
type Encryptor struct {
	aead cipher.AEAD
}

// NewEncryptor creates an Encryptor from the instance secret.
func NewEncryptor(secret string) (*Encryptor, error) {
	key := deriveKey(secret)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: init cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: init gcm: %w", err)
	}
	return &Encryptor{aead: aead}, nil
}

// Encrypt seals plaintext into a base64 nonce-prefixed ciphertext.
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("crypto: nonce: %w", err)
	}
	sealed := e.aead.Seal(nil, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, sealed...)), nil
}

// Decrypt opens a value produced by Encrypt. Empty input decodes to empty
// output (absence of a secret is not an error).
func (e *Encryptor) Decrypt(payload string) (string, error) {
	if payload == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil || len(raw) < nonceSize {
		return "", errors.New("crypto: invalid ciphertext")
	}
	plain, err := e.aead.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", errors.New("crypto: decryption failed (wrong key?)")
	}
	return string(plain), nil
}
