// Concurrent queries example - parallel requests with errgroup.
package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/resolvedb/resolvedb-go"
)

type Weather struct {
	Location string  `json:"location"`
	TempC    float64 `json:"temp_c"`
}

func main() {
	// Client is safe for concurrent use
	client, err := resolvedb.New()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	cities := []string{"tokyo", "paris", "london", "sydney", "quebec"}

	// Use WaitGroup for parallel queries
	var wg sync.WaitGroup
	results := make(map[string]*Weather)
	errors := make(map[string]error)
	var mu sync.Mutex

	ctx := context.Background()

	for _, city := range cities {
		wg.Add(1)
		go func(city string) {
			defer wg.Done()

			var w Weather
			err := client.Get(ctx, "weather", city, &w)

			mu.Lock()
			if err != nil {
				errors[city] = err
			} else {
				results[city] = &w
			}
			mu.Unlock()
		}(city)
	}

	wg.Wait()

	// Print results
	fmt.Println("=== Weather Results ===")
	for _, city := range cities {
		if w, ok := results[city]; ok {
			fmt.Printf("%-10s: %.1f°C\n", w.Location, w.TempC)
		} else if err, ok := errors[city]; ok {
			log.Printf("%-10s: error - %v", city, err)
		}
	}

	fmt.Printf("\nSuccessful: %d/%d\n", len(results), len(cities))
}
