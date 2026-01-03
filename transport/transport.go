// Package transport provides DNS transport implementations for ResolveDB.
package transport

import (
	"context"
	"io"
)

// Transport defines the interface for DNS query transports.
type Transport interface {
	// Name returns the transport name (e.g., "doh", "dot", "dns").
	Name() string

	// Query sends a DNS query and returns the response.
	Query(ctx context.Context, req *Request) (*Response, error)

	// IsEncrypted returns true if the transport uses encryption (TLS/HTTPS).
	IsEncrypted() bool

	// Close releases any resources held by the transport.
	Close() error
}

// Request represents a DNS query request.
type Request struct {
	Name   string   // Query name (FQDN)
	Type   uint16   // Query type (TXT, NULL, etc.)
	Labels []string // Parsed labels for convenience
}

// Response represents a DNS query response.
type Response struct {
	Data    []byte // Raw TXT record data
	TTL     uint32 // TTL from DNS response
	Records [][]byte // Multiple TXT records if present
}

// Common DNS record types.
const (
	TypeA     uint16 = 1
	TypeNS    uint16 = 2
	TypeCNAME uint16 = 5
	TypeSOA   uint16 = 6
	TypePTR   uint16 = 12
	TypeMX    uint16 = 15
	TypeTXT   uint16 = 16
	TypeAAAA  uint16 = 28
	TypeSRV   uint16 = 33
	TypeNULL  uint16 = 10
)

// Closer wraps io.Closer for transports that don't need cleanup.
type noopCloser struct{}

func (noopCloser) Close() error { return nil }

// EmbedCloser can be embedded in transport implementations that don't need Close().
type EmbedCloser struct{ noopCloser }

// Multi wraps multiple transports with automatic fallback.
type Multi struct {
	transports []Transport
}

// NewMulti creates a multi-transport with fallback support.
func NewMulti(transports ...Transport) *Multi {
	return &Multi{transports: transports}
}

func (m *Multi) Name() string {
	if len(m.transports) > 0 {
		return "multi(" + m.transports[0].Name() + "+fallback)"
	}
	return "multi"
}

func (m *Multi) Query(ctx context.Context, req *Request) (*Response, error) {
	var lastErr error
	for _, t := range m.transports {
		resp, err := t.Query(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		// Continue to next transport on error
	}
	return nil, lastErr
}

func (m *Multi) IsEncrypted() bool {
	// Only encrypted if ALL transports are encrypted
	for _, t := range m.transports {
		if !t.IsEncrypted() {
			return false
		}
	}
	return len(m.transports) > 0
}

func (m *Multi) Close() error {
	var errs []error
	for _, t := range m.transports {
		if closer, ok := t.(io.Closer); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// Transports returns the underlying transports.
func (m *Multi) Transports() []Transport {
	return m.transports
}
