// Large data example - chunked data handling for large payloads.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/resolvedb/resolvedb-go"
)

func main() {
	client, err := resolvedb.New(
		resolvedb.WithAPIKey("your-api-key"),
		resolvedb.WithNamespace("data-store"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// For large data that exceeds TXT record limits (255 bytes per string, ~64KB total),
	// ResolveDB automatically chunks the data and stores it as a blob.

	// Example: Store a large configuration
	largeConfig := map[string]any{
		"rules": generateLargeRuleSet(100), // Generate 100 rules
		"metadata": map[string]string{
			"version":    "2.0",
			"created_by": "admin",
		},
	}

	fmt.Println("Storing large configuration...")
	err = client.Set(ctx, "configs", "firewall-rules", largeConfig,
		resolvedb.WithForceBlob(true), // Force blob storage for large data
	)
	if err != nil {
		log.Printf("Store error: %v", err)
		// Note: In production, you'd handle ErrPayloadTooLarge specially
	} else {
		fmt.Println("Configuration stored successfully")
	}

	// Retrieve - SDK automatically handles chunk assembly
	fmt.Println("\nRetrieving large configuration...")
	var retrieved map[string]any
	err = client.Get(ctx, "configs", "firewall-rules", &retrieved)
	if err != nil {
		log.Printf("Retrieve error: %v", err)
	} else {
		if rules, ok := retrieved["rules"].([]any); ok {
			fmt.Printf("Retrieved %d rules\n", len(rules))
		}
	}
}

// generateLargeRuleSet creates example firewall rules.
func generateLargeRuleSet(count int) []map[string]any {
	rules := make([]map[string]any, count)
	for i := 0; i < count; i++ {
		rules[i] = map[string]any{
			"id":       fmt.Sprintf("rule-%d", i+1),
			"action":   "allow",
			"protocol": "tcp",
			"port":     8080 + i,
			"source":   "0.0.0.0/0",
			"comment":  fmt.Sprintf("Auto-generated rule %d for service %d", i+1, i+1),
		}
	}
	return rules
}
