package security

import (
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/hkdf"
)

// DeriveKey derives a key using HKDF-SHA256.
// Per PROTOCOL.md: info = fqdn + clientPubKey + serverPubKey + timestamp + nonce
func DeriveKey(secret, salt, info []byte, length int) ([]byte, error) {
	reader := hkdf.New(sha256.New, secret, salt, info)
	key := make([]byte, length)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// DeriveKey32 derives a 32-byte (256-bit) key.
func DeriveKey32(secret, salt, info []byte) (*[32]byte, error) {
	derived, err := DeriveKey(secret, salt, info, 32)
	if err != nil {
		return nil, err
	}
	var key [32]byte
	copy(key[:], derived)
	return &key, nil
}

// BuildHKDFInfo builds the info parameter for HKDF.
// Format: fqdn|clientPubKey|serverPubKey|timestamp|nonce
func BuildHKDFInfo(fqdn string, clientPubKey, serverPubKey []byte, timestamp int64, nonce []byte) []byte {
	// Simple concatenation with length prefixes for unambiguous parsing
	info := make([]byte, 0, len(fqdn)+len(clientPubKey)+len(serverPubKey)+8+len(nonce)+5*4)

	// FQDN
	info = append(info, byte(len(fqdn)>>8), byte(len(fqdn)))
	info = append(info, []byte(fqdn)...)

	// Client public key
	info = append(info, byte(len(clientPubKey)>>8), byte(len(clientPubKey)))
	info = append(info, clientPubKey...)

	// Server public key
	info = append(info, byte(len(serverPubKey)>>8), byte(len(serverPubKey)))
	info = append(info, serverPubKey...)

	// Timestamp (8 bytes, big-endian)
	info = append(info,
		byte(timestamp>>56), byte(timestamp>>48),
		byte(timestamp>>40), byte(timestamp>>32),
		byte(timestamp>>24), byte(timestamp>>16),
		byte(timestamp>>8), byte(timestamp),
	)

	// Nonce
	info = append(info, byte(len(nonce)>>8), byte(len(nonce)))
	info = append(info, nonce...)

	return info
}
