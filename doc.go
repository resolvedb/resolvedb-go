// Package resolvedb provides a Go client for ResolveDB, a DNS-based data storage system.
//
// ResolveDB enables storing and retrieving data through DNS queries using the Universal
// Query Response Protocol (UQRP). This approach offers unique advantages including
// ubiquitous accessibility, built-in caching through DNS TTLs, and firewall-friendly
// communication.
//
// # Quick Start
//
// The simplest way to get started is with a zero-config client:
//
//	client := resolvedb.New()
//	var weather Weather
//	err := client.Get(context.Background(), "weather", "quebec", &weather)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Temperature: %.1f°F\n", weather.TempF)
//
// # Configuration
//
// Use functional options to configure the client:
//
//	client, err := resolvedb.New(
//	    resolvedb.WithAPIKey("your-api-key"),
//	    resolvedb.WithNamespace("myapp"),
//	    resolvedb.WithTimeout(10*time.Second),
//	)
//
// # Transports
//
// Multiple transport protocols are supported with automatic fallback:
//
//   - DoH (DNS-over-HTTPS) - Default, most reliable
//   - DoH JSON - Google-style JSON API
//   - DoT (DNS-over-TLS) - Encrypted traditional DNS
//   - DNS (UDP/TCP) - Traditional DNS queries
//
// Configure transports with priority order:
//
//	client, err := resolvedb.New(
//	    resolvedb.WithTransports(
//	        transport.NewDoH(),
//	        transport.NewDoT(),
//	        transport.NewDNS(),
//	    ),
//	)
//
// # Security
//
// The SDK supports multiple security patterns:
//
//   - API Key authentication for write operations
//   - AES-256-GCM encryption for sensitive data
//   - BDT (Blind Device Tokens) for IoT devices
//   - CTP (Cohort Token Pattern) for user targeting
//   - NBA (Namespace-Bound Authentication) for multi-tenant apps
//
// # Error Handling
//
// Errors are typed and can be checked with errors.Is:
//
//	err := client.Get(ctx, "data", "key", &result)
//	if errors.Is(err, resolvedb.ErrNotFound) {
//	    // Handle missing data
//	}
//	if errors.Is(err, resolvedb.ErrRateLimited) {
//	    // Back off and retry
//	}
//
// # Thread Safety
//
// The Client is safe for concurrent use from multiple goroutines.
package resolvedb
