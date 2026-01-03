// GeoIP example - IP address geolocation.
package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/resolvedb/resolvedb-go"
	"github.com/resolvedb/resolvedb-go/services/geoip"
)

func main() {
	client, err := resolvedb.New()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	geo := geoip.NewClient(client)
	ctx := context.Background()

	// Lookup various IPs
	ips := []string{
		"8.8.8.8",       // Google DNS
		"1.1.1.1",       // Cloudflare DNS
		"208.67.222.222", // OpenDNS
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		loc, err := geo.Lookup(ctx, ip)
		if err != nil {
			log.Printf("%s: error - %v", ipStr, err)
			continue
		}

		fmt.Printf("%s:\n", ipStr)
		fmt.Printf("  City: %s, %s\n", loc.City, loc.Country)
		fmt.Printf("  Coordinates: %.4f, %.4f\n", loc.Latitude, loc.Longitude)
		if loc.ISP != "" {
			fmt.Printf("  ISP: %s\n", loc.ISP)
		}
		fmt.Println()
	}
}
