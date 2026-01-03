// ML Registry example - distributed model configuration for GPU clusters.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/resolvedb/resolvedb-go"
)

// ModelConfig represents ML model deployment configuration.
type ModelConfig struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	Endpoint   string   `json:"endpoint"`
	GPUType    string   `json:"gpu_type"`
	Replicas   int      `json:"replicas"`
	MaxBatch   int      `json:"max_batch_size"`
	Timeout    int      `json:"timeout_ms"`
	Features   []string `json:"features"`
	Deprecated bool     `json:"deprecated"`
}

func main() {
	client, err := resolvedb.New(
		resolvedb.WithNamespace("ml-platform"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Query model registry for deployment config
	models := []string{"gpt-4-turbo", "embeddings-v3", "whisper-large"}

	fmt.Println("=== ML Model Registry ===")

	for _, modelName := range models {
		var config ModelConfig
		err := client.Get(ctx, "models", modelName, &config)
		if err != nil {
			if resolvedb.IsNotFound(err) {
				fmt.Printf("%s: not registered\n\n", modelName)
			} else {
				log.Printf("%s: error - %v\n\n", modelName, err)
			}
			continue
		}

		fmt.Printf("Model: %s\n", config.Name)
		fmt.Printf("  Version: %s\n", config.Version)
		fmt.Printf("  Endpoint: %s\n", config.Endpoint)
		fmt.Printf("  GPU: %s x %d replicas\n", config.GPUType, config.Replicas)
		fmt.Printf("  Max Batch: %d, Timeout: %dms\n", config.MaxBatch, config.Timeout)
		if config.Deprecated {
			fmt.Printf("  ⚠️  DEPRECATED - migrate to newer version\n")
		}
		fmt.Println()
	}

	// List all available models
	fmt.Println("=== Available Models ===")
	modelList, err := client.List(ctx, "models")
	if err != nil {
		log.Printf("List error: %v", err)
		return
	}

	for _, m := range modelList {
		fmt.Printf("  - %s\n", m)
	}
}
