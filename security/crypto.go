// Package security provides cryptographic operations for ResolveDB.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"sync/atomic"
)

// ErrNonceExhausted is returned when the nonce counter overflows.
var ErrNonceExhausted = errors.New("nonce counter exhausted, rotate encryption key")

// ErrInvalidCiphertext is returned when decryption fails.
var ErrInvalidCiphertext = errors.New("invalid ciphertext")

// AESGCMNonceSize is the standard nonce size for AES-GCM.
const AESGCMNonceSize = 12

// AESGCMTagSize is the authentication tag size for AES-GCM.
const AESGCMTagSize = 16

// EncryptionContext provides AES-256-GCM encryption with nonce tracking.
// Per security review: uses counter-based nonces to prevent reuse.
type EncryptionContext struct {
	key     [32]byte
	counter atomic.Uint64
}

// NewEncryptionContext creates a new encryption context.
func NewEncryptionContext(key []byte) (*EncryptionContext, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}
	ctx := &EncryptionContext{}
	copy(ctx.key[:], key)
	return ctx, nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns: nonce || ciphertext || tag
func (e *EncryptionContext) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key[:])
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	// Generate nonce using counter + random
	nonce, err := e.generateNonce()
	if err != nil {
		return nil, err
	}

	// Encrypt with authenticated data
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Return nonce || ciphertext (tag is appended by Seal)
	result := make([]byte, AESGCMNonceSize+len(ciphertext))
	copy(result[:AESGCMNonceSize], nonce)
	copy(result[AESGCMNonceSize:], ciphertext)

	return result, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM.
// Input format: nonce || ciphertext || tag
func (e *EncryptionContext) Decrypt(data []byte) ([]byte, error) {
	if len(data) < AESGCMNonceSize+AESGCMTagSize {
		return nil, ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(e.key[:])
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	nonce := data[:AESGCMNonceSize]
	ciphertext := data[AESGCMNonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	return plaintext, nil
}

// generateNonce creates a unique nonce using counter + randomness.
func (e *EncryptionContext) generateNonce() ([]byte, error) {
	counter := e.counter.Add(1)
	if counter == 0 {
		// Counter overflow - nonce space exhausted
		return nil, ErrNonceExhausted
	}

	nonce := make([]byte, AESGCMNonceSize)

	// First 8 bytes: counter (big-endian)
	nonce[0] = byte(counter >> 56)
	nonce[1] = byte(counter >> 48)
	nonce[2] = byte(counter >> 40)
	nonce[3] = byte(counter >> 32)
	nonce[4] = byte(counter >> 24)
	nonce[5] = byte(counter >> 16)
	nonce[6] = byte(counter >> 8)
	nonce[7] = byte(counter)

	// Last 4 bytes: random for additional entropy
	if _, err := rand.Read(nonce[8:]); err != nil {
		return nil, fmt.Errorf("generate random: %w", err)
	}

	return nonce, nil
}

// ZeroKey securely zeros the encryption key.
// Note: Go's GC may have already copied the key elsewhere.
// For highly sensitive applications, consider using memguard.
func (e *EncryptionContext) ZeroKey() {
	for i := range e.key {
		e.key[i] = 0
	}
}

// Encrypt encrypts plaintext with the given key using AES-256-GCM.
// This is a convenience function for one-off encryption.
// Uses fully random nonces (safe for standalone calls, no counter tracking).
func Encrypt(plaintext []byte, key *[32]byte) ([]byte, error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	// Use fully random nonce for standalone encryption
	// This is safe because each call generates a new random nonce
	nonce := make([]byte, AESGCMNonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, AESGCMNonceSize+len(ciphertext))
	copy(result[:AESGCMNonceSize], nonce)
	copy(result[AESGCMNonceSize:], ciphertext)

	return result, nil
}

// Decrypt decrypts ciphertext with the given key using AES-256-GCM.
// This is a convenience function for one-off decryption.
func Decrypt(ciphertext []byte, key *[32]byte) ([]byte, error) {
	ctx, err := NewEncryptionContext(key[:])
	if err != nil {
		return nil, err
	}
	return ctx.Decrypt(ciphertext)
}

// GenerateKey generates a random 256-bit encryption key.
func GenerateKey() (*[32]byte, error) {
	key := new([32]byte)
	if _, err := rand.Read(key[:]); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	return key, nil
}
