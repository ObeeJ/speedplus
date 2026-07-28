// Package crypto provides field-level encryption for PII that must never be
// stored in plaintext — recipient names/phone numbers for package delivery.
// Bootstrap key management: a single 32-byte key from ENCRYPTION_KEY. Migrate
// to a KMS-managed key later without a schema change (ciphertext format is
// independent of where the key comes from).
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var ErrInvalidKeySize = errors.New("encryption key must be exactly 32 bytes")

// Cipher wraps an AES-256-GCM key for encrypting/decrypting short PII strings.
type Cipher struct {
	gcm cipher.AEAD
}

// NewCipher builds a Cipher from a raw 32-byte key.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	return &Cipher{gcm: gcm}, nil
}

// Encrypt returns a base64(nonce || ciphertext || tag) string. Empty input
// encrypts to an empty string so optional fields round-trip cleanly.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	sealed := c.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. Empty input decrypts to an empty string.
func (c *Cipher) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64: %w", err)
	}
	nonceSize := c.gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, sealed := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := c.gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}

// EncryptPtr/DecryptPtr are convenience wrappers for the *string fields used
// throughout the order/recipient models — nil in, nil out.
func (c *Cipher) EncryptPtr(s *string) (*string, error) {
	if s == nil {
		return nil, nil
	}
	enc, err := c.Encrypt(*s)
	if err != nil {
		return nil, err
	}
	return &enc, nil
}

func (c *Cipher) DecryptPtr(s *string) (*string, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	dec, err := c.Decrypt(*s)
	if err != nil {
		return nil, err
	}
	return &dec, nil
}
