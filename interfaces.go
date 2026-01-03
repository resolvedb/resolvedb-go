package resolvedb

import "context"

// Querier provides read operations on ResolveDB.
type Querier interface {
	// Get retrieves data for a resource and key, unmarshaling into dst.
	Get(ctx context.Context, resource, key string, dst any, opts ...RequestOption) error

	// GetRaw retrieves raw data for a resource and key.
	GetRaw(ctx context.Context, resource, key string, opts ...RequestOption) (*Response, error)

	// List retrieves a list of keys for a resource.
	List(ctx context.Context, resource string, opts ...RequestOption) ([]string, error)
}

// Writer provides write operations on ResolveDB.
type Writer interface {
	// Set stores data for a resource and key.
	Set(ctx context.Context, resource, key string, data any, opts ...RequestOption) error

	// Delete removes data for a resource and key.
	Delete(ctx context.Context, resource, key string, opts ...RequestOption) error
}

// ReadWriter combines read and write operations.
type ReadWriter interface {
	Querier
	Writer
}

// EncryptedQuerier provides encrypted read operations.
type EncryptedQuerier interface {
	// GetEncrypted retrieves and decrypts data.
	GetEncrypted(ctx context.Context, resource, key string, dst any, opts ...RequestOption) error
}

// EncryptedWriter provides encrypted write operations.
type EncryptedWriter interface {
	// SetEncrypted encrypts and stores data.
	SetEncrypted(ctx context.Context, resource, key string, data any, opts ...RequestOption) error
}

// SecureClient combines all secure operations.
type SecureClient interface {
	ReadWriter
	EncryptedQuerier
	EncryptedWriter
}

// Ensure Client implements all interfaces.
var (
	_ Querier          = (*Client)(nil)
	_ Writer           = (*Client)(nil)
	_ ReadWriter       = (*Client)(nil)
	_ EncryptedQuerier = (*Client)(nil)
	_ EncryptedWriter  = (*Client)(nil)
	_ SecureClient     = (*Client)(nil)
)
