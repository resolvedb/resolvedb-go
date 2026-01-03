// Multi-tenant example - NBA signatures for namespace isolation.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/resolvedb/resolvedb-go"
	"github.com/resolvedb/resolvedb-go/security"
)

// TenantConfig represents per-tenant configuration.
type TenantConfig struct {
	MaxUsers       int      `json:"max_users"`
	Features       []string `json:"features"`
	CustomDomain   string   `json:"custom_domain,omitempty"`
	AllowedRegions []string `json:"allowed_regions"`
}

func main() {
	// Tenant signing key (shared between client and server)
	signingKey := []byte("tenant-signing-key-32-bytes!!!!!")

	// Create client for tenant "acme-corp"
	client, err := resolvedb.New(
		resolvedb.WithNamespace("acme-corp"),
		resolvedb.WithTenantQueryKey(signingKey),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Generate NBA signature for this query
	nba, err := security.NewNBA("acme-corp", "config", "settings", signingKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("NBA Signature: %s\n\n", nba.String())

	// Query with NBA - cryptographically proves namespace binding
	var config TenantConfig
	err = client.Get(ctx, "config", "settings", &config,
		resolvedb.WithNBA(nba.String()),
	)
	if err != nil {
		log.Printf("Config fetch error: %v", err)
		// Use defaults
		config = TenantConfig{
			MaxUsers:       100,
			Features:       []string{"basic"},
			AllowedRegions: []string{"us-east-1"},
		}
	}

	fmt.Println("Tenant Configuration (acme-corp):")
	fmt.Printf("  Max Users: %d\n", config.MaxUsers)
	fmt.Printf("  Features: %v\n", config.Features)
	fmt.Printf("  Custom Domain: %s\n", config.CustomDomain)
	fmt.Printf("  Allowed Regions: %v\n", config.AllowedRegions)
}
