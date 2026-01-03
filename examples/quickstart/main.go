// Quickstart example - get weather data with zero configuration.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/resolvedb/resolvedb-go"
)

// Weather represents weather data.
type Weather struct {
	Location   string  `json:"location"`
	TempC      float64 `json:"temp_c"`
	TempF      float64 `json:"temp_f"`
	Conditions string  `json:"conditions"`
}

func main() {
	// Zero-config client - no API key needed for public data
	client, err := resolvedb.New()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Get weather for Quebec
	var weather Weather
	err = client.Get(context.Background(), "weather", "quebec", &weather)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Weather in %s:\n", weather.Location)
	fmt.Printf("  Temperature: %.1f°C (%.1f°F)\n", weather.TempC, weather.TempF)
	fmt.Printf("  Conditions: %s\n", weather.Conditions)
}
