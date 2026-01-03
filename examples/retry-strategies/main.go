// Retry strategies example - exponential backoff with jitter.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/resolvedb/resolvedb-go"
)

func main() {
	// Configure custom retry behavior
	client, err := resolvedb.New(
		resolvedb.WithRetry(resolvedb.RetryConfig{
			MaxRetries:     5,                      // Up to 5 retries
			InitialBackoff: 100 * time.Millisecond, // Start with 100ms
			MaxBackoff:     30 * time.Second,       // Cap at 30s
			Multiplier:     2.0,                    // Double each time
			JitterFactor:   0.2,                    // ±20% jitter
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Query with automatic retry on transient errors
	// Retryable errors: E010 (server error), E012 (timeout), E013 (rate limited)
	type Data struct {
		Value string `json:"value"`
	}

	var data Data
	err = client.Get(ctx, "config", "settings", &data)
	if err != nil {
		// Check if it's a retryable error that exhausted retries
		if resolvedb.IsRetryable(err) {
			log.Printf("Request failed after retries: %v", err)
		} else if errors.Is(err, resolvedb.ErrNotFound) {
			log.Printf("Resource not found (not retryable)")
		} else {
			log.Printf("Request failed: %v", err)
		}
		return
	}

	fmt.Printf("Retrieved: %s\n", data.Value)
}
