package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Token prefixes per PROTOCOL.md.
const (
	PrefixBDT = "bdt-"
	PrefixCTP = "ctp-"
	PrefixNBA = "sig-"
)

// BDT (Blind Device Token) provides anonymous device identity.
// Format: bdt-<32-hex-chars>
// Use case: IoT devices querying config without revealing identity.
// Rotation: Weekly recommended.
type BDT struct {
	token string
}

// NewBDT creates a new Blind Device Token.
func NewBDT() (*BDT, error) {
	bytes := make([]byte, 16) // 128 bits
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("generate random: %w", err)
	}
	return &BDT{token: PrefixBDT + hex.EncodeToString(bytes)}, nil
}

// NewBDTFromString creates a BDT from an existing token string.
// Validates format: must be "bdt-" followed by exactly 32 hex characters.
func NewBDTFromString(token string) (*BDT, error) {
	// Check minimum length (prefix + 32 hex chars)
	if len(token) != len(PrefixBDT)+32 {
		return nil, fmt.Errorf("invalid BDT format: expected %d chars, got %d", len(PrefixBDT)+32, len(token))
	}

	// Check prefix
	if token[:len(PrefixBDT)] != PrefixBDT {
		return nil, fmt.Errorf("invalid BDT format: must start with %q", PrefixBDT)
	}

	// Validate hex characters
	hexPart := token[len(PrefixBDT):]
	if _, err := hex.DecodeString(hexPart); err != nil {
		return nil, fmt.Errorf("invalid BDT format: %w", err)
	}

	return &BDT{token: token}, nil
}

// String returns the token string.
func (b *BDT) String() string {
	return b.token
}

// CTP (Cohort Token Pattern) enables user targeting without exposing identity.
// Format: ctp-<encrypted-base64>
// Contains: encrypted user ID, cohort info, timestamp, nonce.
type CTP struct {
	token string
}

// CTPPayload is the encrypted payload for CTP tokens.
type CTPPayload struct {
	UserID    string `json:"uid"`
	Cohort    string `json:"coh,omitempty"`
	Timestamp int64  `json:"ts"`
	Nonce     string `json:"nonce"`
}

// NewCTP creates a new Cohort Token Pattern token.
// The payload is encrypted with the provided key.
func NewCTP(userID, cohort string, key *[32]byte) (*CTP, error) {
	// Generate nonce for replay protection
	nonceBytes := make([]byte, 8)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	payload := CTPPayload{
		UserID:    userID,
		Cohort:    cohort,
		Timestamp: time.Now().Unix(),
		Nonce:     hex.EncodeToString(nonceBytes),
	}

	// JSON encode
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	// Encrypt
	encrypted, err := Encrypt(data, key)
	if err != nil {
		return nil, fmt.Errorf("encrypt: %w", err)
	}

	// Base64 encode
	encoded := base64.RawURLEncoding.EncodeToString(encrypted)

	return &CTP{token: PrefixCTP + encoded}, nil
}

// String returns the token string.
func (c *CTP) String() string {
	return c.token
}

// ValidateCTP validates and decrypts a CTP token.
// Returns the payload if valid, error otherwise.
// Per security review: 30-second replay window.
func ValidateCTP(token string, key *[32]byte) (*CTPPayload, error) {
	if len(token) < len(PrefixCTP) {
		return nil, fmt.Errorf("invalid CTP format")
	}

	encoded := token[len(PrefixCTP):]
	encrypted, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	var payload CTPPayload
	if err := json.Unmarshal(decrypted, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	// Check timestamp (30-second window per security review)
	now := time.Now().Unix()
	if payload.Timestamp < now-30 || payload.Timestamp > now+30 {
		return nil, fmt.Errorf("token expired or future-dated")
	}

	return &payload, nil
}

// NBA (Namespace-Bound Authentication) cryptographically binds queries to namespaces.
// Format: sig-<32-hex-chars>-t-<unix-timestamp>
// Per security review: 128-bit signatures to prevent birthday attacks.
type NBA struct {
	signature string
	timestamp int64
}

// NewNBA creates a new Namespace-Bound Authentication signature.
func NewNBA(namespace, resource, key string, signingKey []byte) (*NBA, error) {
	timestamp := time.Now().Unix()

	// Build message: namespace|resource|key|timestamp
	message := fmt.Sprintf("%s|%s|%s|%d", namespace, resource, key, timestamp)

	// HMAC-SHA256
	mac := hmac.New(sha256.New, signingKey)
	mac.Write([]byte(message))
	signature := mac.Sum(nil)

	// Use first 16 bytes (128 bits) per security review
	sig := hex.EncodeToString(signature[:16])

	return &NBA{
		signature: fmt.Sprintf("%s%s-t-%d", PrefixNBA, sig, timestamp),
		timestamp: timestamp,
	}, nil
}

// String returns the signature string.
func (n *NBA) String() string {
	return n.signature
}

// ValidateNBA validates an NBA signature.
// Per security review: constant-time comparison.
func ValidateNBA(token, namespace, resource, key string, signingKey []byte, maxAge time.Duration) error {
	// Parse token
	if len(token) < len(PrefixNBA)+32 {
		return fmt.Errorf("invalid NBA format")
	}

	// Extract signature and timestamp
	parts := token[len(PrefixNBA):]
	idx := len(parts) - 1
	for idx >= 0 && parts[idx] != '-' {
		idx--
	}
	if idx < 3 || parts[idx-2:idx] != "-t" {
		return fmt.Errorf("invalid NBA format: missing timestamp")
	}

	sigHex := parts[:idx-2]
	tsStr := parts[idx+1:]

	timestamp, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp")
	}

	// Check timestamp
	now := time.Now().Unix()
	if timestamp < now-int64(maxAge.Seconds()) || timestamp > now+30 {
		return fmt.Errorf("signature expired or future-dated")
	}

	// Reconstruct expected signature
	message := fmt.Sprintf("%s|%s|%s|%d", namespace, resource, key, timestamp)
	mac := hmac.New(sha256.New, signingKey)
	mac.Write([]byte(message))
	expected := mac.Sum(nil)
	expectedHex := hex.EncodeToString(expected[:16])

	// Constant-time comparison per security review
	if !ConstantTimeCompare([]byte(sigHex), []byte(expectedHex)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}
