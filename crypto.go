package resolvedb

import (
	"github.com/resolvedb/resolvedb-go/security"
)

// encrypt encrypts data using AES-256-GCM.
func encrypt(plaintext []byte, key *[32]byte) ([]byte, error) {
	return security.Encrypt(plaintext, key)
}

// decrypt decrypts data using AES-256-GCM.
func decrypt(ciphertext []byte, key *[32]byte) ([]byte, error) {
	return security.Decrypt(ciphertext, key)
}

// GenerateEncryptionKey generates a random 256-bit encryption key.
func GenerateEncryptionKey() ([]byte, error) {
	key, err := security.GenerateKey()
	if err != nil {
		return nil, err
	}
	return key[:], nil
}
