// Package geoip provides a client for ResolveDB's GeoIP service.
package geoip

import (
	"context"
	"net"

	"github.com/resolvedb/resolvedb-go"
)

// GeoIPClient defines the interface for GeoIP operations.
// Implement this interface for testing with mocks.
type GeoIPClient interface {
	Lookup(ctx context.Context, ip net.IP, opts ...resolvedb.RequestOption) (*Location, error)
	LookupString(ctx context.Context, ip string, opts ...resolvedb.RequestOption) (*Location, error)
	LookupSelf(ctx context.Context, opts ...resolvedb.RequestOption) (*Location, error)
}

// Client is a GeoIP service client.
type Client struct {
	client resolvedb.Querier
}

// NewClient creates a new GeoIP client.
func NewClient(c resolvedb.Querier) *Client {
	return &Client{client: c}
}

// Ensure Client implements GeoIPClient.
var _ GeoIPClient = (*Client)(nil)

// Location represents a geographic location.
type Location struct {
	IP          string  `json:"ip"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
	ISP         string  `json:"isp,omitempty"`
	ASN         int     `json:"asn,omitempty"`
	ASOrg       string  `json:"as_org,omitempty"`
}

// Lookup retrieves geolocation data for an IP address.
//
// Example:
//
//	loc, err := geoClient.Lookup(ctx, net.ParseIP("8.8.8.8"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("City: %s, Country: %s\n", loc.City, loc.Country)
func (c *Client) Lookup(ctx context.Context, ip net.IP, opts ...resolvedb.RequestOption) (*Location, error) {
	var loc Location
	err := c.client.Get(ctx, "geoip", ip.String(), &loc, opts...)
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

// LookupString retrieves geolocation data for an IP address string.
func (c *Client) LookupString(ctx context.Context, ip string, opts ...resolvedb.RequestOption) (*Location, error) {
	var loc Location
	err := c.client.Get(ctx, "geoip", ip, &loc, opts...)
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

// LookupSelf retrieves geolocation data for the client's IP address.
func (c *Client) LookupSelf(ctx context.Context, opts ...resolvedb.RequestOption) (*Location, error) {
	return c.LookupString(ctx, "self", opts...)
}
