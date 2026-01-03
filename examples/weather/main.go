// Weather service example - various ways to query weather data.
package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/resolvedb/resolvedb-go"
	"github.com/resolvedb/resolvedb-go/services/weather"
)

func main() {
	client, err := resolvedb.New()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	wx := weather.NewClient(client)
	ctx := context.Background()

	// By city name
	fmt.Println("=== By City ===")
	w, err := wx.ByCity(ctx, "tokyo")
	if err != nil {
		log.Printf("Tokyo weather error: %v", err)
	} else {
		fmt.Printf("Tokyo: %.1f°C, %s\n", w.TempC, w.Conditions)
	}

	// By coordinates
	fmt.Println("\n=== By Coordinates ===")
	w, err = wx.ByCoords(ctx, 40.7128, -74.0060) // New York
	if err != nil {
		log.Printf("NYC weather error: %v", err)
	} else {
		fmt.Printf("NYC (40.71, -74.01): %.1f°C, %s\n", w.TempC, w.Conditions)
	}

	// By IP address
	fmt.Println("\n=== By IP ===")
	w, err = wx.ByIP(ctx, net.ParseIP("8.8.8.8"))
	if err != nil {
		log.Printf("IP weather error: %v", err)
	} else {
		fmt.Printf("8.8.8.8 location: %.1f°C, %s\n", w.TempC, w.Conditions)
	}
}
