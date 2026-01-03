// Package weather provides a client for ResolveDB's Weather service.
package weather

import (
	"context"
	"fmt"
	"net"

	"github.com/resolvedb/resolvedb-go"
)

// WeatherClient defines the interface for Weather operations.
// Implement this interface for testing with mocks.
type WeatherClient interface {
	ByCity(ctx context.Context, city string, opts ...resolvedb.RequestOption) (*Weather, error)
	ByCoords(ctx context.Context, lat, lon float64, opts ...resolvedb.RequestOption) (*Weather, error)
	ByIP(ctx context.Context, ip net.IP, opts ...resolvedb.RequestOption) (*Weather, error)
	BySelf(ctx context.Context, opts ...resolvedb.RequestOption) (*Weather, error)
}

// Client is a Weather service client.
type Client struct {
	client resolvedb.Querier
}

// NewClient creates a new Weather client.
func NewClient(c resolvedb.Querier) *Client {
	return &Client{client: c}
}

// Ensure Client implements WeatherClient.
var _ WeatherClient = (*Client)(nil)

// Weather represents current weather conditions.
type Weather struct {
	Location    string  `json:"location"`
	Temperature float64 `json:"temperature"`
	TempC       float64 `json:"temp_c"`
	TempF       float64 `json:"temp_f"`
	FeelsLike   float64 `json:"feels_like"`
	FeelsLikeC  float64 `json:"feels_like_c"`
	FeelsLikeF  float64 `json:"feels_like_f"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"wind_speed"`
	WindDir     string  `json:"wind_dir"`
	Conditions  string  `json:"conditions"`
	Icon        string  `json:"icon,omitempty"`
	Sunrise     string  `json:"sunrise,omitempty"`
	Sunset      string  `json:"sunset,omitempty"`
	UpdatedAt   string  `json:"updated_at,omitempty"`
}

// Forecast represents a weather forecast entry.
type Forecast struct {
	Date       string  `json:"date"`
	TempHighC  float64 `json:"temp_high_c"`
	TempHighF  float64 `json:"temp_high_f"`
	TempLowC   float64 `json:"temp_low_c"`
	TempLowF   float64 `json:"temp_low_f"`
	Conditions string  `json:"conditions"`
	Icon       string  `json:"icon,omitempty"`
}

// ByCity retrieves weather for a city.
//
// Example:
//
//	weather, err := wxClient.ByCity(ctx, "quebec")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Temperature: %.1f°C\n", weather.TempC)
func (c *Client) ByCity(ctx context.Context, city string, opts ...resolvedb.RequestOption) (*Weather, error) {
	var w Weather
	err := c.client.Get(ctx, "weather", city, &w, opts...)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// ByCoords retrieves weather for coordinates.
//
// Example:
//
//	weather, err := wxClient.ByCoords(ctx, 46.81, -71.21)  // Quebec City
func (c *Client) ByCoords(ctx context.Context, lat, lon float64, opts ...resolvedb.RequestOption) (*Weather, error) {
	key := fmt.Sprintf("%.4f,%.4f", lat, lon)
	var w Weather
	err := c.client.Get(ctx, "weather", key, &w, opts...)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// ByIP retrieves weather for an IP address location.
func (c *Client) ByIP(ctx context.Context, ip net.IP, opts ...resolvedb.RequestOption) (*Weather, error) {
	var w Weather
	err := c.client.Get(ctx, "weather", "ip-"+ip.String(), &w, opts...)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// BySelf retrieves weather for the client's location.
func (c *Client) BySelf(ctx context.Context, opts ...resolvedb.RequestOption) (*Weather, error) {
	return c.ByCity(ctx, "self", opts...)
}
