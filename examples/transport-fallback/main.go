// Transport fallback example - DoH → DoT → DNS with automatic failover.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/resolvedb/resolvedb-go"
	"github.com/resolvedb/resolvedb-go/transport"
)

func main() {
	// Configure multiple transports with priority order
	// First tries DoH, then DoT, then traditional DNS
	client, err := resolvedb.New(
		resolvedb.WithTransports(
			transport.NewDoH(),                    // Primary: DNS-over-HTTPS
			transport.NewDoT(),                    // Fallback 1: DNS-over-TLS
			transport.NewDNS(),                    // Fallback 2: Traditional DNS (unencrypted)
		),
		resolvedb.WithTimeout(10*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// This query will try each transport in order until one succeeds
	type Weather struct {
		Location   string  `json:"location"`
		TempC      float64 `json:"temp_c"`
		Conditions string  `json:"conditions"`
	}

	var weather Weather
	err = client.Get(ctx, "weather", "paris", &weather)
	if err != nil {
		log.Fatalf("All transports failed: %v", err)
	}

	fmt.Printf("Weather in %s: %.1f°C, %s\n",
		weather.Location, weather.TempC, weather.Conditions)
	fmt.Println("\nTransport fallback worked!")
}
