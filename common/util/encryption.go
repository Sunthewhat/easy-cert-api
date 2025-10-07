package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

// EncryptData encrypts data using AES-256-GCM
// keyHex: 64-character hex string (32 bytes)
// Returns: base64-encoded string containing [nonce][ciphertext][auth_tag]
func EncryptData(data []byte, keyHex string) (string, error) {
	// Decode hex key to bytes
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", errors.New("invalid encryption key format")
	}

	if len(key) != 32 {
		return "", errors.New("encryption key must be 32 bytes (64 hex chars)")
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize()) // 12 bytes for GCM
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt data (output includes auth tag)
	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptData decrypts data using AES-256-GCM
// encryptedData: base64-encoded string containing [nonce][ciphertext][auth_tag]
// keyHex: 64-character hex string (32 bytes)
func DecryptData(encryptedData string, keyHex string) ([]byte, error) {
	// Decode hex key to bytes
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, errors.New("invalid encryption key format")
	}

	if len(key) != 32 {
		return nil, errors.New("encryption key must be 32 bytes (64 hex chars)")
	}

	// Decode base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// Extract nonce and encrypted data
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and verify auth tag
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: data may be tampered")
	}

	return plaintext, nil
}
