// Basic example - CRUD operations with authentication.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/resolvedb/resolvedb-go"
)

// UserPrefs represents user preferences.
type UserPrefs struct {
	Theme       string   `json:"theme"`
	Language    string   `json:"language"`
	Timezone    string   `json:"timezone"`
	Notifications bool   `json:"notifications"`
	Features    []string `json:"features"`
}

func main() {
	// Create authenticated client
	client, err := resolvedb.New(
		resolvedb.WithAPIKey("your-api-key"),
		resolvedb.WithNamespace("myapp"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	ctx := context.Background()

	// CREATE - Store data
	fmt.Println("=== Create ===")
	prefs := UserPrefs{
		Theme:       "dark",
		Language:    "en-US",
		Timezone:    "America/New_York",
		Notifications: true,
		Features:    []string{"beta", "analytics"},
	}

	err = client.Set(ctx, "preferences", "user-123", prefs)
	if err != nil {
		log.Printf("Create error: %v", err)
	} else {
		fmt.Println("User preferences created")
	}

	// READ - Retrieve data
	fmt.Println("\n=== Read ===")
	var retrieved UserPrefs
	err = client.Get(ctx, "preferences", "user-123", &retrieved)
	if err != nil {
		log.Printf("Read error: %v", err)
	} else {
		fmt.Printf("Theme: %s\n", retrieved.Theme)
		fmt.Printf("Language: %s\n", retrieved.Language)
		fmt.Printf("Features: %v\n", retrieved.Features)
	}

	// UPDATE - Modify data
	fmt.Println("\n=== Update ===")
	retrieved.Theme = "light"
	retrieved.Features = append(retrieved.Features, "ai-assist")

	err = client.Set(ctx, "preferences", "user-123", retrieved)
	if err != nil {
		log.Printf("Update error: %v", err)
	} else {
		fmt.Println("User preferences updated")
	}

	// DELETE - Remove data
	fmt.Println("\n=== Delete ===")
	err = client.Delete(ctx, "preferences", "user-123")
	if err != nil {
		log.Printf("Delete error: %v", err)
	} else {
		fmt.Println("User preferences deleted")
	}

	// Verify deletion
	fmt.Println("\n=== Verify Deletion ===")
	err = client.Get(ctx, "preferences", "user-123", &retrieved)
	if resolvedb.IsNotFound(err) {
		fmt.Println("Confirmed: preferences no longer exist")
	} else if err != nil {
		log.Printf("Unexpected error: %v", err)
	}
}
