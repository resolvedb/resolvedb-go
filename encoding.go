package resolvedb

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Encoding prefixes used in DNS labels.
// Per RFC 1035, colons are invalid in DNS labels, so hyphens are used.
const (
	PrefixBase64 = "b64-"
	PrefixHex    = "hex-"
	PrefixAuth   = "auth-"
	PrefixBDT    = "bdt-"
	PrefixCTP    = "ctp-"
	PrefixSig    = "sig-"
)

// encodeBase64 encodes data as URL-safe base64 without padding.
func encodeBase64(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// decodeBase64 decodes URL-safe base64 data (with or without padding).
func decodeBase64(s string) ([]byte, error) {
	// Try without padding first
	data, err := base64.RawURLEncoding.DecodeString(s)
	if err == nil {
		return data, nil
	}
	// Try with padding
	return base64.URLEncoding.DecodeString(s)
}

// encodeHex encodes data as lowercase hexadecimal.
func encodeHex(data []byte) string {
	return hex.EncodeToString(data)
}

// decodeHex decodes hexadecimal data.
func decodeHex(s string) ([]byte, error) {
	return hex.DecodeString(strings.ToLower(s))
}

// encodeJSON marshals data to JSON and then base64 encodes it.
func encodeJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return encodeBase64(data), nil
}

// decodeJSON base64 decodes and unmarshals JSON data.
func decodeJSON(s string, v any) error {
	data, err := decodeBase64(s)
	if err != nil {
		return fmt.Errorf("base64 decode: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json unmarshal: %w", err)
	}
	return nil
}

// encodeParam encodes a parameter value for use in a DNS label.
// Uses base64 for binary data, hex for short values.
func encodeParam(data []byte) string {
	if len(data) <= 16 {
		return PrefixHex + encodeHex(data)
	}
	return PrefixBase64 + encodeBase64(data)
}

// decodeParam decodes a parameter value from a DNS label.
func decodeParam(s string) ([]byte, error) {
	switch {
	case strings.HasPrefix(s, PrefixBase64):
		return decodeBase64(strings.TrimPrefix(s, PrefixBase64))
	case strings.HasPrefix(s, PrefixHex):
		return decodeHex(strings.TrimPrefix(s, PrefixHex))
	default:
		// Plain text parameter
		return []byte(s), nil
	}
}

// sanitizeLabel ensures a string is valid for use in a DNS label.
// Converts to lowercase, replaces invalid characters.
func sanitizeLabel(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		} else if r == '_' || r == ' ' {
			result.WriteRune('-')
		}
	}
	// DNS labels must start and end with alphanumeric
	label := result.String()
	label = strings.Trim(label, "-")
	// Max label length is 63 characters
	if len(label) > 63 {
		label = label[:63]
	}
	return label
}
