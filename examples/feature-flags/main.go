// Feature flags example - kill switches and gradual rollouts.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/resolvedb/resolvedb-go"
	"github.com/resolvedb/resolvedb-go/services/flags"
)

func main() {
	// For authenticated flag management, use API key
	client, err := resolvedb.New(
		resolvedb.WithNamespace("myapp"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	flagClient := flags.NewClient(client)
	ctx := context.Background()

	// Check simple boolean flag
	fmt.Println("=== Boolean Flags ===")
	if enabled, err := flagClient.Get(ctx, "dark-mode"); err != nil {
		log.Printf("dark-mode error: %v", err)
	} else if enabled {
		fmt.Println("Dark mode is enabled")
	} else {
		fmt.Println("Dark mode is disabled")
	}

	// Get with default (graceful degradation)
	fmt.Println("\n=== Flags with Defaults ===")
	paymentsV2 := flagClient.GetWithDefault(ctx, "payments-v2", false)
	fmt.Printf("Payments V2: %v\n", paymentsV2)

	// Check full flag configuration
	fmt.Println("\n=== Full Flag Config ===")
	flag, err := flagClient.GetFull(ctx, "new-checkout")
	if err != nil {
		log.Printf("new-checkout error: %v", err)
	} else {
		fmt.Printf("Flag: %s\n", flag.Name)
		fmt.Printf("  Enabled: %v\n", flag.Enabled)
		fmt.Printf("  Percentage: %d%%\n", flag.Percentage)
		fmt.Printf("  Cohorts: %v\n", flag.Cohorts)
	}
}
