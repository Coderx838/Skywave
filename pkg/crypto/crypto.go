package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// DeriveKey derives a 32-byte AES key from a room secret passphrase using SHA-256.
func DeriveKey(passphrase string) []byte {
	hash := sha256.Sum256([]byte("skywave-e2ee-salt:" + passphrase))
	return hash[:]
}

// Encrypt encrypts plaintext using AES-256-GCM with the derived key.
// Returns base64-encoded string: nonce + ciphertext.
func Encrypt(plaintext string, passphrase string) (string, error) {
	key := DeriveKey(passphrase)
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

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM and passphrase.
func Decrypt(encoded string, passphrase string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	key := DeriveKey(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	plainBytes, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", errors.New("failed to decrypt message: invalid passphrase or corrupted data")
	}

	return string(plainBytes), nil
}

// HashSecret produces a cryptographic SHA-256 hash for verifying room or server access.
func HashSecret(secret string, salt string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", salt, secret)))
	return fmt.Sprintf("%x", sum)
}
