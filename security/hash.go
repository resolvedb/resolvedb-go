package security

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// SHA256 computes the SHA-256 hash of data.
func SHA256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// SHA256Hex computes the SHA-256 hash and returns it as a hex string.
func SHA256Hex(data []byte) string {
	return hex.EncodeToString(SHA256(data))
}

// ConstantTimeCompare compares two byte slices in constant time.
// Per security review: prevents timing attacks on token validation.
func ConstantTimeCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// ConstantTimeCompareString compares two strings in constant time.
func ConstantTimeCompareString(a, b string) bool {
	return ConstantTimeCompare([]byte(a), []byte(b))
}

// VerifyHash verifies that data matches the expected hash.
func VerifyHash(data []byte, expectedHex string) bool {
	actual := SHA256Hex(data)
	return ConstantTimeCompareString(actual, expectedHex)
}

// VerifyChunkIntegrity verifies the integrity of a data chunk.
// Per security review: verify per-chunk hash before assembly.
func VerifyChunkIntegrity(chunk []byte, expectedHash string) error {
	if !VerifyHash(chunk, expectedHash) {
		return ErrChunkIntegrity
	}
	return nil
}

// ErrChunkIntegrity is returned when chunk integrity verification fails.
var ErrChunkIntegrity = ErrInvalidCiphertext // Reuse error to not leak info
