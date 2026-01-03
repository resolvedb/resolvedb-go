# ResolveDB Go SDK

Official Go SDK for [ResolveDB](https://resolvedb.io) - DNS-based data storage system.

[![Go Reference](https://pkg.go.dev/badge/github.com/resolvedb/resolvedb-go.svg)](https://pkg.go.dev/github.com/resolvedb/resolvedb-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/resolvedb/resolvedb-go)](https://goreportcard.com/report/github.com/resolvedb/resolvedb-go)

## TL;DR

```go
client, _ := resolvedb.New()
var weather Weather
_ = client.Get(context.Background(), "weather", "quebec", &weather)
fmt.Printf("Quebec: %.1f°F\n", weather.TempF)
```

## Installation

```bash
go get github.com/resolvedb/resolvedb-go
```

Requires Go 1.21+

## Quick Start

### Zero-Config (Public Data)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/resolvedb/resolvedb-go"
)

type Weather struct {
    Location string  `json:"location"`
    TempC    float64 `json:"temp_c"`
}

func main() {
    client, err := resolvedb.New()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    var weather Weather
    err = client.Get(context.Background(), "weather", "tokyo", &weather)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%s: %.1f°C\n", weather.Location, weather.TempC)
}
```

### Authenticated Client

```go
client, err := resolvedb.New(
    resolvedb.WithAPIKey("your-api-key"),
    resolvedb.WithNamespace("myapp"),
)
if err != nil {
    log.Fatal(err)
}
```

## Why ResolveDB?

| Feature | Traditional API | ResolveDB |
|---------|-----------------|-----------|
| Firewall traversal | Blocked ports | DNS always works |
| Built-in caching | Implement yourself | DNS TTL caching |
| Global distribution | CDN setup required | DNS infrastructure |
| Latency | HTTP overhead | DNS optimized |
| Protocol | REST/GraphQL | Universal Query Response Protocol |

## Core Operations

### Get / Set / Delete

```go
// Get data
var config Config
err := client.Get(ctx, "config", "app-settings", &config)

// Set data (requires API key)
err := client.Set(ctx, "config", "app-settings", myConfig)

// Delete data (requires API key)
err := client.Delete(ctx, "config", "app-settings")
```

### List Resources

```go
keys, err := client.List(ctx, "config")
for _, key := range keys {
    fmt.Println(key)
}
```

## Configuration Options

```go
client := resolvedb.New(
    resolvedb.WithAPIKey("key"),           // API key for writes
    resolvedb.WithNamespace("myapp"),      // Tenant namespace
    resolvedb.WithTimeout(10*time.Second), // Request timeout
    resolvedb.WithTransports(              // Transport priority
        transport.NewDoH(),
        transport.NewDoT(),
    ),
    resolvedb.WithRetry(resolvedb.RetryConfig{
        MaxRetries:     3,
        InitialBackoff: 100 * time.Millisecond,
    }),
    resolvedb.WithCache(resolvedb.CacheConfig{
        Enabled:    true,
        MaxEntries: 1000,
    }),
)
```

## Transport Options

| Transport | Security | Use Case |
|-----------|----------|----------|
| `DoH` | HTTPS | Default, most reliable |
| `DoH JSON` | HTTPS | Google-style JSON API |
| `DoT` | TLS | Encrypted DNS |
| `DNS` | None | Traditional (not for auth) |

```go
// Multi-transport with fallback
client := resolvedb.New(
    resolvedb.WithTransports(
        transport.NewDoH(),  // Try first
        transport.NewDoT(),  // Fallback
        transport.NewDNS(),  // Last resort
    ),
)
```

## Service Clients

### Weather

```go
import "github.com/resolvedb/resolvedb-go/services/weather"

wx := weather.NewClient(client)
w, _ := wx.ByCity(ctx, "paris")
w, _ := wx.ByCoords(ctx, 48.8566, 2.3522)
```

### GeoIP

```go
import "github.com/resolvedb/resolvedb-go/services/geoip"

geo := geoip.NewClient(client)
loc, _ := geo.Lookup(ctx, net.ParseIP("8.8.8.8"))
fmt.Printf("City: %s\n", loc.City)
```

### Feature Flags

```go
import "github.com/resolvedb/resolvedb-go/services/flags"

flagClient := flags.NewClient(client)
if enabled, _ := flagClient.Get(ctx, "dark-mode"); enabled {
    enableDarkMode()
}
```

## Security Features

### Client-Side Encryption (AES-256-GCM)

```go
client := resolvedb.New(
    resolvedb.WithAPIKey("key"),
    resolvedb.WithEncryptionKey(myKey),
)

// Encrypt before storing
err := client.SetEncrypted(ctx, "secrets", "api-keys", secrets)

// Decrypt when retrieving
err := client.GetEncrypted(ctx, "secrets", "api-keys", &secrets)
```

### Security Tokens

```go
import "github.com/resolvedb/resolvedb-go/security"

// BDT - Blind Device Tokens (IoT)
bdt, _ := security.NewBDT()
client.Get(ctx, "config", "device", &cfg, resolvedb.WithBDT(bdt.String()))

// NBA - Namespace-Bound Authentication
nba, _ := security.NewNBA("tenant", "resource", "key", signingKey)
client.Get(ctx, "data", "key", &data, resolvedb.WithNBA(nba.String()))

// CTP - Cohort Token Pattern
ctp, _ := security.NewCTP("user-id", "cohort", encKey)
```

## Error Handling

```go
err := client.Get(ctx, "data", "key", &result)

// Type checking
if errors.Is(err, resolvedb.ErrNotFound) {
    // Resource doesn't exist
}
if errors.Is(err, resolvedb.ErrRateLimited) {
    // Back off and retry
}
if errors.Is(err, resolvedb.ErrUnauthorized) {
    // Auth required
}

// Check if retryable
if resolvedb.IsRetryable(err) {
    // Safe to retry (E010, E012, E013)
}
```

### Error Codes

| Code | Name | Retryable |
|------|------|-----------|
| E001 | Bad Request | No |
| E002 | Unauthorized | No |
| E003 | Forbidden | No |
| E004 | Not Found | No |
| E005 | Conflict | No |
| E006 | Payload Too Large | No |
| E010 | Server Error | Yes |
| E012 | Timeout | Yes |
| E013 | Rate Limited | Yes |
| E014 | Encryption Required | No |

## Thread Safety

The `Client` is safe for concurrent use from multiple goroutines:

```go
client := resolvedb.New()

var wg sync.WaitGroup
for _, city := range cities {
    wg.Add(1)
    go func(city string) {
        defer wg.Done()
        var w Weather
        client.Get(ctx, "weather", city, &w)
    }(city)
}
wg.Wait()
```

## Testing

Use interfaces for easy mocking:

```go
type WeatherService struct {
    client resolvedb.Querier  // Interface, not concrete type
}

// In tests
mock := &MockClient{}
service := &WeatherService{client: mock}
```

## Examples

See the [examples](./examples) directory:

- `quickstart/` - Zero-config quick start
- `basic/` - CRUD operations
- `weather/` - Weather service client
- `geoip/` - IP geolocation
- `feature-flags/` - Kill switches and rollouts
- `iot-config/` - IoT device configuration with BDT
- `encrypted/` - AES-256-GCM encryption
- `multi-tenant/` - NBA signatures
- `transport-fallback/` - DoH → DoT → DNS fallback
- `retry-strategies/` - Exponential backoff
- `context-timeout/` - Cancellation handling
- `concurrent/` - Parallel queries
- `testing/` - Interface-based mocking
- `ml-registry/` - ML model configuration
- `large-data/` - Chunked data handling

## API Reference

Full documentation: [pkg.go.dev/github.com/resolvedb/resolvedb-go](https://pkg.go.dev/github.com/resolvedb/resolvedb-go)

## License

MIT License - see [LICENSE](LICENSE) for details.
