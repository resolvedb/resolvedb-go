package resolvedb

import (
	"fmt"
	"net/http"
	"time"

	"github.com/resolvedb/resolvedb-go/transport"
)

// Option configures a Client.
type Option func(*clientConfig)

// clientConfig holds client configuration.
type clientConfig struct {
	apiKey          string
	namespace       string
	version         string
	tld             string
	baseURL         string
	transports      []transport.Transport
	timeout         time.Duration
	retryConfig     RetryConfig
	cacheConfig     CacheConfig
	encryptionKey   *[32]byte
	tenantQueryKey  []byte
	httpClient      *http.Client
	enforceSecurity bool
}

// defaultConfig returns the default client configuration.
func defaultConfig() *clientConfig {
	return &clientConfig{
		version:         "v1",
		tld:             "net",
		baseURL:         "https://api.resolvedb.io",
		timeout:         30 * time.Second,
		retryConfig:     DefaultRetryConfig(),
		cacheConfig:     DefaultCacheConfig(),
		enforceSecurity: true,
	}
}

// WithAPIKey sets the API key for authenticated operations.
func WithAPIKey(key string) Option {
	return func(c *clientConfig) {
		c.apiKey = key
	}
}

// WithNamespace sets the namespace for multi-tenant operations.
func WithNamespace(ns string) Option {
	return func(c *clientConfig) {
		c.namespace = ns
	}
}

// WithVersion sets the protocol version (default: "v1").
func WithVersion(v string) Option {
	return func(c *clientConfig) {
		c.version = v
	}
}

// WithTLD sets the TLD for queries (default: "net").
func WithTLD(tld string) Option {
	return func(c *clientConfig) {
		c.tld = tld
	}
}

// WithBaseURL sets the DoH endpoint URL (default: "https://api.resolvedb.io").
func WithBaseURL(url string) Option {
	return func(c *clientConfig) {
		c.baseURL = url
	}
}

// WithTransports sets the transport priority order with automatic fallback.
// The first transport is tried first; on failure, subsequent transports are tried.
func WithTransports(transports ...transport.Transport) Option {
	return func(c *clientConfig) {
		c.transports = transports
	}
}

// WithTimeout sets the request timeout (default: 30s).
func WithTimeout(d time.Duration) Option {
	return func(c *clientConfig) {
		c.timeout = d
	}
}

// WithRetry configures retry behavior.
func WithRetry(config RetryConfig) Option {
	return func(c *clientConfig) {
		c.retryConfig = config
	}
}

// WithCache configures response caching.
func WithCache(config CacheConfig) Option {
	return func(c *clientConfig) {
		c.cacheConfig = config
	}
}

// WithEncryptionKey sets the AES-256-GCM encryption key for encrypted operations.
// The key must be exactly 32 bytes. Panics if the key length is invalid.
func WithEncryptionKey(key []byte) Option {
	if len(key) != 32 {
		panic(fmt.Sprintf("resolvedb: encryption key must be 32 bytes, got %d", len(key)))
	}
	return func(c *clientConfig) {
		var k [32]byte
		copy(k[:], key)
		c.encryptionKey = &k
	}
}

// WithTenantQueryKey sets the key for NBA (Namespace-Bound Authentication) signatures.
func WithTenantQueryKey(key []byte) Option {
	return func(c *clientConfig) {
		c.tenantQueryKey = make([]byte, len(key))
		copy(c.tenantQueryKey, key)
	}
}

// WithHTTPClient sets a custom HTTP client for DoH transport.
func WithHTTPClient(client *http.Client) Option {
	return func(c *clientConfig) {
		c.httpClient = client
	}
}

// WithoutSecurityEnforcement disables security enforcement (NOT RECOMMENDED).
// By default, authenticated requests are blocked on unencrypted transports.
// Only disable this for testing or when using a trusted network.
func WithoutSecurityEnforcement() Option {
	return func(c *clientConfig) {
		c.enforceSecurity = false
	}
}

// RequestOption configures a single request.
type RequestOption func(*requestConfig)

// requestConfig holds per-request configuration.
type requestConfig struct {
	ttl        time.Duration
	forceBlob  bool
	skipCache  bool
	encrypt    bool
	bdtToken   string
	ctpToken   string
	nbaToken   string
}

// WithTTL sets the TTL for a write operation.
func WithTTL(d time.Duration) RequestOption {
	return func(c *requestConfig) {
		c.ttl = d
	}
}

// WithForceBlob forces data to be stored as a blob, bypassing TXT record limits.
func WithForceBlob(force bool) RequestOption {
	return func(c *requestConfig) {
		c.forceBlob = force
	}
}

// WithSkipCache bypasses the cache for this request.
func WithSkipCache() RequestOption {
	return func(c *requestConfig) {
		c.skipCache = true
	}
}

// WithEncrypt enables encryption for this request.
func WithEncrypt() RequestOption {
	return func(c *requestConfig) {
		c.encrypt = true
	}
}

// WithBDT sets the Blind Device Token for this request.
func WithBDT(token string) RequestOption {
	return func(c *requestConfig) {
		c.bdtToken = token
	}
}

// WithCTP sets the Cohort Token Pattern token for this request.
func WithCTP(token string) RequestOption {
	return func(c *requestConfig) {
		c.ctpToken = token
	}
}

// WithNBA sets the Namespace-Bound Authentication signature for this request.
func WithNBA(signature string) RequestOption {
	return func(c *requestConfig) {
		c.nbaToken = signature
	}
}
