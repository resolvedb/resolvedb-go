// Encrypted data example - client-side AES-256-GCM encryption.
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/resolvedb/resolvedb-go"
	"github.com/resolvedb/resolvedb-go/security"
)

// Secret represents sensitive configuration.
type Secret struct {
	APIKey      string `json:"api_key"`
	DatabaseURL string `json:"database_url"`
	JWTSecret   string `json:"jwt_secret"`
}

func main() {
	// Generate encryption key (in production, load from secure storage)
	encKey, err := security.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Encryption key (store securely): %s\n\n", hex.EncodeToString(encKey[:]))

	// Create client with encryption key and API key for writes
	client, err := resolvedb.New(
		resolvedb.WithAPIKey("your-api-key"),
		resolvedb.WithNamespace("myapp"),
		resolvedb.WithEncryptionKey(encKey[:]),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// Store encrypted secret
	secret := Secret{
		APIKey:      "sk-prod-abc123",
		DatabaseURL: "postgres://user:pass@host/db",
		JWTSecret:   "super-secret-jwt-key",
	}

	fmt.Println("Storing encrypted secret...")
	err = client.SetEncrypted(ctx, "secrets", "production", secret)
	if err != nil {
		log.Fatalf("Store error: %v", err)
	}
	fmt.Println("Secret stored successfully")

	// Retrieve and decrypt
	fmt.Println("Retrieving encrypted secret...")
	var retrieved Secret
	err = client.GetEncrypted(ctx, "secrets", "production", &retrieved)
	if err != nil {
		log.Fatalf("Retrieve error: %v", err)
	}

	fmt.Println("Retrieved Secret:")
	fmt.Printf("  API Key: %s\n", retrieved.APIKey)
	fmt.Printf("  Database URL: %s\n", retrieved.DatabaseURL)
	fmt.Printf("  JWT Secret: %s\n", retrieved.JWTSecret)
}
