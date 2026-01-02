package finalride

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// GenerateKey generates a random 32-byte AES-256 key
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptData encrypts data using AES-GCM encryption
func EncryptData(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	return append(nonce, ciphertext...), nil
}

// DecryptData decrypts data using AES-GCM encryption
func DecryptData(encryptedData []byte, key []byte) ([]byte, error) {
	if len(encryptedData) < 12 {
		return nil, fmt.Errorf("encrypted data too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := encryptedData[:12]
	ciphertext := encryptedData[12:]

	return aesgcm.Open(nil, nonce, ciphertext, nil)
}
