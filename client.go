package resolvedb

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/resolvedb/resolvedb-go/transport"
)

// Client is a ResolveDB client.
// It is safe for concurrent use from multiple goroutines.
type Client struct {
	config    *clientConfig
	transport transport.Transport
	cache     Cache
}

// New creates a new ResolveDB client with the given options.
//
// Example:
//
//	// Zero-config client for public data
//	client, err := resolvedb.New()
//
//	// Authenticated client
//	client, err := resolvedb.New(
//	    resolvedb.WithAPIKey("your-api-key"),
//	    resolvedb.WithNamespace("myapp"),
//	)
func New(opts ...Option) (*Client, error) {
	config := defaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set up transport
	var t transport.Transport
	if len(config.transports) > 0 {
		if len(config.transports) == 1 {
			t = config.transports[0]
		} else {
			t = transport.NewMulti(config.transports...)
		}
	} else {
		// Default to DoH with configured options
		dohOpts := []transport.DoHOption{
			transport.WithDoHURL(config.baseURL + "/dns-query"),
		}
		if config.httpClient != nil {
			dohOpts = append(dohOpts, transport.WithDoHClient(config.httpClient))
		} else if config.timeout > 0 {
			// Create HTTP client with configured timeout
			dohOpts = append(dohOpts, transport.WithDoHClient(&http.Client{
				Timeout: config.timeout,
			}))
		}
		t = transport.NewDoH(dohOpts...)
	}

	// Set up cache
	var cache Cache
	if config.cacheConfig.Enabled {
		cache = newMemoryCache(config.cacheConfig)
	} else {
		cache = noopCache{}
	}

	return &Client{
		config:    config,
		transport: t,
		cache:     cache,
	}, nil
}

// MustNew creates a new ResolveDB client with the given options.
// Panics if the configuration is invalid.
// Use New() for error handling in production code.
func MustNew(opts ...Option) *Client {
	client, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return client
}

// validateConfig validates the client configuration.
func validateConfig(config *clientConfig) error {
	if config.version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	if config.tld == "" {
		return fmt.Errorf("TLD cannot be empty")
	}
	if config.timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}
	return nil
}

// Get retrieves data for a resource and key, unmarshaling into dst.
//
// Example:
//
//	var weather Weather
//	err := client.Get(ctx, "weather", "quebec", &weather)
func (c *Client) Get(ctx context.Context, resource, key string, dst any, opts ...RequestOption) error {
	resp, err := c.GetRaw(ctx, resource, key, opts...)
	if err != nil {
		return err
	}
	return resp.Unmarshal(dst)
}

// GetRaw retrieves raw response data for a resource and key.
func (c *Client) GetRaw(ctx context.Context, resource, key string, opts ...RequestOption) (*Response, error) {
	reqConfig := &requestConfig{}
	for _, opt := range opts {
		opt(reqConfig)
	}

	// Build query name
	queryName := c.buildQueryName("get", resource, key, reqConfig)

	// Check cache
	cacheKey := buildCacheKey("get", resource, key, c.config.namespace, c.config.version)
	if !reqConfig.skipCache {
		if cached, ok := c.cache.Get(cacheKey); ok {
			return cached, nil
		}
	}

	// Execute query with retry
	resp, err := doWithRetry(ctx, c.config.retryConfig, func() (*Response, error) {
		return c.executeQuery(ctx, queryName, reqConfig)
	})
	if err != nil {
		return nil, err
	}

	// Cache successful responses
	if resp.IsSuccess() && !reqConfig.skipCache {
		c.cache.Set(cacheKey, resp, resp.TTL)
	}

	return resp, nil
}

// Set stores data for a resource and key.
//
// Example:
//
//	err := client.Set(ctx, "config", "settings", myConfig,
//	    resolvedb.WithTTL(24*time.Hour),
//	)
func (c *Client) Set(ctx context.Context, resource, key string, data any, opts ...RequestOption) error {
	if c.config.apiKey == "" {
		return ErrUnauthorized
	}

	reqConfig := &requestConfig{}
	for _, opt := range opts {
		opt(reqConfig)
	}

	// Security check: authenticated requests require encrypted transport
	if c.config.enforceSecurity && !c.transport.IsEncrypted() {
		return ErrEncryptedTransportRequired
	}

	// Encode data
	encoded, err := encodeJSON(data)
	if err != nil {
		return fmt.Errorf("encode data: %w", err)
	}

	// Build query name
	queryName := c.buildQueryNameWithData("put", resource, key, encoded, reqConfig)

	// Execute query
	resp, err := doWithRetry(ctx, c.config.retryConfig, func() (*Response, error) {
		return c.executeQuery(ctx, queryName, reqConfig)
	})
	if err != nil {
		return err
	}

	if err := resp.ToError(); err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := buildCacheKey("get", resource, key, c.config.namespace, c.config.version)
	c.cache.Delete(cacheKey)

	return nil
}

// Delete removes data for a resource and key.
func (c *Client) Delete(ctx context.Context, resource, key string, opts ...RequestOption) error {
	if c.config.apiKey == "" {
		return ErrUnauthorized
	}

	reqConfig := &requestConfig{}
	for _, opt := range opts {
		opt(reqConfig)
	}

	// Security check
	if c.config.enforceSecurity && !c.transport.IsEncrypted() {
		return ErrEncryptedTransportRequired
	}

	queryName := c.buildQueryName("delete", resource, key, reqConfig)

	resp, err := doWithRetry(ctx, c.config.retryConfig, func() (*Response, error) {
		return c.executeQuery(ctx, queryName, reqConfig)
	})
	if err != nil {
		return err
	}

	if err := resp.ToError(); err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := buildCacheKey("get", resource, key, c.config.namespace, c.config.version)
	c.cache.Delete(cacheKey)

	return nil
}

// List retrieves a list of keys for a resource.
func (c *Client) List(ctx context.Context, resource string, opts ...RequestOption) ([]string, error) {
	reqConfig := &requestConfig{}
	for _, opt := range opts {
		opt(reqConfig)
	}

	queryName := c.buildQueryName("list", resource, "", reqConfig)

	resp, err := doWithRetry(ctx, c.config.retryConfig, func() (*Response, error) {
		return c.executeQuery(ctx, queryName, reqConfig)
	})
	if err != nil {
		return nil, err
	}

	if err := resp.ToError(); err != nil {
		return nil, err
	}

	var keys []string
	if err := resp.Unmarshal(&keys); err != nil {
		return nil, err
	}

	return keys, nil
}

// GetEncrypted retrieves and decrypts data.
func (c *Client) GetEncrypted(ctx context.Context, resource, key string, dst any, opts ...RequestOption) error {
	if c.config.encryptionKey == nil {
		return fmt.Errorf("encryption key not configured")
	}

	opts = append(opts, WithEncrypt())
	resp, err := c.GetRaw(ctx, resource, key, opts...)
	if err != nil {
		return err
	}

	// Decrypt data
	decrypted, err := decrypt(resp.Data, c.config.encryptionKey)
	if err != nil {
		return fmt.Errorf("decrypt: %w", err)
	}

	// Create new response with decrypted data
	decryptedResp := *resp
	decryptedResp.Data = decrypted
	return decryptedResp.Unmarshal(dst)
}

// SetEncrypted encrypts and stores data.
func (c *Client) SetEncrypted(ctx context.Context, resource, key string, data any, opts ...RequestOption) error {
	if c.config.encryptionKey == nil {
		return fmt.Errorf("encryption key not configured")
	}

	// Encode data
	encoded, err := encodeJSON(data)
	if err != nil {
		return fmt.Errorf("encode data: %w", err)
	}

	// Encrypt
	encrypted, err := encrypt([]byte(encoded), c.config.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	// Store encrypted data
	opts = append(opts, WithEncrypt())
	reqConfig := &requestConfig{}
	for _, opt := range opts {
		opt(reqConfig)
	}

	if c.config.enforceSecurity && !c.transport.IsEncrypted() {
		return ErrEncryptedTransportRequired
	}

	queryName := c.buildQueryNameWithData("put", resource, key, encodeBase64(encrypted), reqConfig)

	resp, err := doWithRetry(ctx, c.config.retryConfig, func() (*Response, error) {
		return c.executeQuery(ctx, queryName, reqConfig)
	})
	if err != nil {
		return err
	}

	return resp.ToError()
}

// Close releases resources held by the client.
func (c *Client) Close() error {
	return c.transport.Close()
}

// buildQueryName builds the FQDN for a query.
// Format: <operation>.<params>.<resource>.<namespace>.<version>.resolvedb.<tld>
func (c *Client) buildQueryName(operation, resource, key string, reqConfig *requestConfig) string {
	parts := []string{operation}

	// Add key if present
	if key != "" {
		parts = append(parts, sanitizeLabel(key))
	}

	// Add resource
	parts = append(parts, sanitizeLabel(resource))

	// Add namespace if configured
	if c.config.namespace != "" {
		parts = append(parts, sanitizeLabel(c.config.namespace))
	} else {
		parts = append(parts, "public")
	}

	// Add version
	parts = append(parts, c.config.version)

	// Add TLD
	parts = append(parts, "resolvedb", c.config.tld)

	// Add signed auth token if present (HMAC-signed, not raw API key)
	if c.config.apiKey != "" {
		// Generate time-limited HMAC signature instead of exposing raw API key
		// Format: auth-<signature>-t-<timestamp>
		authToken := c.generateAuthToken(operation, resource, key)
		newParts := []string{parts[0], authToken}
		newParts = append(newParts, parts[1:]...)
		parts = newParts
	}

	// Add security tokens if present
	if reqConfig.bdtToken != "" {
		parts = insertAfter(parts, 0, reqConfig.bdtToken)
	}
	if reqConfig.ctpToken != "" {
		parts = insertAfter(parts, 0, reqConfig.ctpToken)
	}
	if reqConfig.nbaToken != "" {
		parts = insertAfter(parts, 0, reqConfig.nbaToken)
	}

	return strings.Join(parts, ".")
}

// buildQueryNameWithData builds the FQDN for a write query with data.
func (c *Client) buildQueryNameWithData(operation, resource, key, data string, reqConfig *requestConfig) string {
	parts := []string{operation}

	// Add encoded data
	parts = append(parts, PrefixBase64+data)

	// Add key
	parts = append(parts, sanitizeLabel(key))

	// Add resource
	parts = append(parts, sanitizeLabel(resource))

	// Add namespace
	if c.config.namespace != "" {
		parts = append(parts, sanitizeLabel(c.config.namespace))
	} else {
		parts = append(parts, "public")
	}

	// Add version
	parts = append(parts, c.config.version)

	// Add TLD
	parts = append(parts, "resolvedb", c.config.tld)

	// Add signed auth token (HMAC-signed, not raw API key)
	if c.config.apiKey != "" {
		authToken := c.generateAuthToken(operation, resource, key)
		newParts := []string{parts[0], authToken}
		newParts = append(newParts, parts[1:]...)
		parts = newParts
	}

	return strings.Join(parts, ".")
}

// executeQuery sends a DNS query and parses the response.
func (c *Client) executeQuery(ctx context.Context, queryName string, reqConfig *requestConfig) (*Response, error) {
	// Create transport request
	req := &transport.Request{
		Name:   queryName,
		Type:   transport.TypeTXT,
		Labels: strings.Split(queryName, "."),
	}

	// Execute query
	transportResp, err := c.transport.Query(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("transport query: %w", err)
	}

	// Parse UQRP response
	resp, err := ParseResponse(string(transportResp.Data))
	if err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Override TTL from DNS if not set in response
	if resp.TTL == 0 && transportResp.TTL > 0 {
		resp.TTL = time.Duration(transportResp.TTL) * time.Second
	}

	return resp, nil
}

// generateAuthToken creates a time-limited HMAC signature for authentication.
// This prevents exposing the raw API key in DNS queries.
// Format: auth-<signature>-t-<timestamp>
func (c *Client) generateAuthToken(operation, resource, key string) string {
	timestamp := time.Now().Unix()

	// Build message: operation|resource|key|namespace|timestamp
	message := fmt.Sprintf("%s|%s|%s|%s|%d",
		operation, resource, key, c.config.namespace, timestamp)

	// HMAC-SHA256 with API key
	mac := hmac.New(sha256.New, []byte(c.config.apiKey))
	mac.Write([]byte(message))
	signature := mac.Sum(nil)

	// Use first 16 bytes (128 bits) - secure and fits in DNS label
	sig := hex.EncodeToString(signature[:16])

	return fmt.Sprintf("%s%s-t-%d", PrefixAuth, sig, timestamp)
}

// insertAfter inserts a value after the given index.
func insertAfter(slice []string, index int, value string) []string {
	result := make([]string, len(slice)+1)
	copy(result, slice[:index+1])
	result[index+1] = value
	copy(result[index+2:], slice[index+1:])
	return result
}
