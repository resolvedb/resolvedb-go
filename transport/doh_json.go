package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DoHJSON implements DNS-over-HTTPS using JSON API format.
// This follows the Google/Cloudflare JSON API style.
type DoHJSON struct {
	baseURL    string
	httpClient *http.Client
}

// DoHJSONOption configures a DoHJSON transport.
type DoHJSONOption func(*DoHJSON)

// WithDoHJSONURL sets the JSON API endpoint URL.
func WithDoHJSONURL(url string) DoHJSONOption {
	return func(d *DoHJSON) {
		d.baseURL = url
	}
}

// WithDoHJSONClient sets a custom HTTP client.
func WithDoHJSONClient(client *http.Client) DoHJSONOption {
	return func(d *DoHJSON) {
		d.httpClient = client
	}
}

// NewDoHJSON creates a new JSON API DoH transport.
func NewDoHJSON(opts ...DoHJSONOption) *DoHJSON {
	d := &DoHJSON{
		baseURL: "https://api.resolvedb.io/resolve",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *DoHJSON) Name() string { return "doh-json" }

func (d *DoHJSON) IsEncrypted() bool { return true }

func (d *DoHJSON) Close() error { return nil }

// Query sends a DNS query using JSON API.
func (d *DoHJSON) Query(ctx context.Context, req *Request) (*Response, error) {
	// Build URL with query parameters
	u, err := url.Parse(d.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	q.Set("name", req.Name)
	q.Set("type", strconv.Itoa(int(req.Type)))
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/dns-json")

	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return parseJSONResponse(body)
}

// jsonDNSResponse represents the JSON API response format.
type jsonDNSResponse struct {
	Status   int  `json:"Status"`
	TC       bool `json:"TC"`
	RD       bool `json:"RD"`
	RA       bool `json:"RA"`
	AD       bool `json:"AD"`
	CD       bool `json:"CD"`
	Question []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
	} `json:"Question"`
	Answer []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
		TTL  int    `json:"TTL"`
		Data string `json:"data"`
	} `json:"Answer"`
	Authority []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
		TTL  int    `json:"TTL"`
		Data string `json:"data"`
	} `json:"Authority"`
}

// parseJSONResponse parses a JSON API DNS response.
func parseJSONResponse(data []byte) (*Response, error) {
	var jsonResp jsonDNSResponse
	if err := json.Unmarshal(data, &jsonResp); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	resp := &Response{}

	for _, answer := range jsonResp.Answer {
		// Remove surrounding quotes from TXT records
		data := answer.Data
		if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
			data = data[1 : len(data)-1]
		}

		resp.Records = append(resp.Records, []byte(data))
		if resp.TTL == 0 {
			resp.TTL = uint32(answer.TTL)
		}
	}

	// Combine all records
	for _, r := range resp.Records {
		resp.Data = append(resp.Data, r...)
	}

	return resp, nil
}
