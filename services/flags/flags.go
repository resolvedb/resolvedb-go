// Package flags provides a client for ResolveDB's Feature Flags service.
package flags

import (
	"context"

	"github.com/resolvedb/resolvedb-go"
)

// FlagsClient defines the interface for Feature Flags operations.
// Implement this interface for testing with mocks.
type FlagsClient interface {
	Get(ctx context.Context, name string, opts ...resolvedb.RequestOption) (bool, error)
	GetWithDefault(ctx context.Context, name string, defaultValue bool, opts ...resolvedb.RequestOption) bool
	GetFull(ctx context.Context, name string, opts ...resolvedb.RequestOption) (*Flag, error)
	GetValue(ctx context.Context, name string, opts ...resolvedb.RequestOption) (interface{}, error)
	IsEnabledForCohort(ctx context.Context, name, cohort string, opts ...resolvedb.RequestOption) (bool, error)
}

// Client is a Feature Flags service client.
type Client struct {
	client resolvedb.Querier
}

// NewClient creates a new Feature Flags client.
func NewClient(c resolvedb.Querier) *Client {
	return &Client{client: c}
}

// Ensure Client implements FlagsClient.
var _ FlagsClient = (*Client)(nil)

// Flag represents a feature flag.
type Flag struct {
	Name        string      `json:"name"`
	Enabled     bool        `json:"enabled"`
	Value       interface{} `json:"value,omitempty"`
	Percentage  int         `json:"percentage,omitempty"`
	Cohorts     []string    `json:"cohorts,omitempty"`
	Description string      `json:"description,omitempty"`
}

// Get retrieves a feature flag by name.
//
// Example:
//
//	enabled, err := flagClient.Get(ctx, "dark-mode")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if enabled {
//	    enableDarkMode()
//	}
func (c *Client) Get(ctx context.Context, name string, opts ...resolvedb.RequestOption) (bool, error) {
	var flag Flag
	err := c.client.Get(ctx, "flags", name, &flag, opts...)
	if err != nil {
		// Treat not found as disabled
		if resolvedb.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return flag.Enabled, nil
}

// GetWithDefault retrieves a flag with a default value.
func (c *Client) GetWithDefault(ctx context.Context, name string, defaultValue bool, opts ...resolvedb.RequestOption) bool {
	enabled, err := c.Get(ctx, name, opts...)
	if err != nil {
		return defaultValue
	}
	return enabled
}

// GetFull retrieves the complete flag configuration.
func (c *Client) GetFull(ctx context.Context, name string, opts ...resolvedb.RequestOption) (*Flag, error) {
	var flag Flag
	err := c.client.Get(ctx, "flags", name, &flag, opts...)
	if err != nil {
		return nil, err
	}
	return &flag, nil
}

// GetValue retrieves a flag's value (for non-boolean flags).
func (c *Client) GetValue(ctx context.Context, name string, opts ...resolvedb.RequestOption) (interface{}, error) {
	flag, err := c.GetFull(ctx, name, opts...)
	if err != nil {
		return nil, err
	}
	return flag.Value, nil
}

// IsEnabledForCohort checks if a flag is enabled for a specific cohort.
func (c *Client) IsEnabledForCohort(ctx context.Context, name, cohort string, opts ...resolvedb.RequestOption) (bool, error) {
	// Use CTP token if provided via options
	flag, err := c.GetFull(ctx, name, opts...)
	if err != nil {
		return false, err
	}

	if !flag.Enabled {
		return false, nil
	}

	// Check cohort membership
	for _, co := range flag.Cohorts {
		if co == cohort || co == "*" {
			return true, nil
		}
	}

	return false, nil
}
