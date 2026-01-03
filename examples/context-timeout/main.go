// Context timeout example - cancellation and deadline handling.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/resolvedb/resolvedb-go"
)

func main() {
	client, err := resolvedb.New()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Example 1: Context with timeout
	fmt.Println("=== Timeout Example ===")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	type Weather struct {
		Location string  `json:"location"`
		TempC    float64 `json:"temp_c"`
	}

	var weather Weather
	err = client.Get(ctx, "weather", "london", &weather)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("Request timed out after 5 seconds")
		} else {
			log.Printf("Request failed: %v", err)
		}
	} else {
		fmt.Printf("London: %.1f°C\n", weather.TempC)
	}

	// Example 2: Context with cancellation
	fmt.Println("\n=== Cancellation Example ===")
	ctx2, cancel2 := context.WithCancel(context.Background())

	// Simulate cancellation after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel2()
		fmt.Println("Request cancelled!")
	}()

	err = client.Get(ctx2, "weather", "sydney", &weather)
	if err != nil {
		if ctx2.Err() == context.Canceled {
			log.Printf("Request was cancelled")
		} else {
			log.Printf("Request failed: %v", err)
		}
	} else {
		fmt.Printf("Sydney: %.1f°C\n", weather.TempC)
	}

	// Example 3: Deadline propagation
	fmt.Println("\n=== Deadline Example ===")
	deadline := time.Now().Add(3 * time.Second)
	ctx3, cancel3 := context.WithDeadline(context.Background(), deadline)
	defer cancel3()

	err = client.Get(ctx3, "weather", "berlin", &weather)
	if err != nil {
		log.Printf("Request failed: %v", err)
	} else {
		fmt.Printf("Berlin: %.1f°C\n", weather.TempC)
	}
}
