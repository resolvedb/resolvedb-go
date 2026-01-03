package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DoH implements DNS-over-HTTPS transport (RFC 8484).
type DoH struct {
	baseURL    string
	httpClient *http.Client
}

// DoHOption configures a DoH transport.
type DoHOption func(*DoH)

// WithDoHURL sets the DoH endpoint URL.
func WithDoHURL(url string) DoHOption {
	return func(d *DoH) {
		d.baseURL = url
	}
}

// WithDoHClient sets a custom HTTP client.
func WithDoHClient(client *http.Client) DoHOption {
	return func(d *DoH) {
		d.httpClient = client
	}
}

// NewDoH creates a new DoH transport.
func NewDoH(opts ...DoHOption) *DoH {
	d := &DoH{
		baseURL: "https://api.resolvedb.io/dns-query",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *DoH) Name() string { return "doh" }

func (d *DoH) IsEncrypted() bool { return true }

func (d *DoH) Close() error { return nil }

// Query sends a DNS query over HTTPS.
func (d *DoH) Query(ctx context.Context, req *Request) (*Response, error) {
	// Build DNS wire format message
	wireMsg := buildDNSQuery(req.Name, req.Type)

	// RFC 8484: POST with application/dns-message
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL, bytes.NewReader(wireMsg))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/dns-message")
	httpReq.Header.Set("Accept", "application/dns-message")

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

	return parseDNSResponse(body)
}

// QueryGET uses GET method with base64url-encoded query (alternative method).
func (d *DoH) QueryGET(ctx context.Context, req *Request) (*Response, error) {
	wireMsg := buildDNSQuery(req.Name, req.Type)
	encoded := base64.RawURLEncoding.EncodeToString(wireMsg)

	url := fmt.Sprintf("%s?dns=%s", d.baseURL, encoded)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/dns-message")

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

	return parseDNSResponse(body)
}

// buildDNSQuery creates a DNS wire format query message.
func buildDNSQuery(name string, qtype uint16) []byte {
	var buf bytes.Buffer

	// Transaction ID - cryptographically random to prevent cache poisoning
	txid := make([]byte, 2)
	if _, err := rand.Read(txid); err != nil {
		// Fallback to less secure but functional value
		txid = []byte{0x00, 0x01}
	}
	buf.Write(txid)

	// Flags: standard query, recursion desired
	buf.Write([]byte{0x01, 0x00})

	// Question count: 1
	buf.Write([]byte{0x00, 0x01})

	// Answer, Authority, Additional counts: 0
	buf.Write([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	// Question section
	// Encode name as DNS labels
	for _, label := range splitLabels(name) {
		if len(label) > 0 {
			buf.WriteByte(byte(len(label)))
			buf.WriteString(label)
		}
	}
	buf.WriteByte(0x00) // Root label

	// Query type
	buf.WriteByte(byte(qtype >> 8))
	buf.WriteByte(byte(qtype & 0xFF))

	// Query class (IN)
	buf.Write([]byte{0x00, 0x01})

	return buf.Bytes()
}

// parseDNSResponse parses a DNS wire format response.
func parseDNSResponse(data []byte) (*Response, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("response too short")
	}

	// Skip header to answers
	// Header: 12 bytes
	// Questions: variable
	offset := 12

	// Skip question section
	qdcount := int(data[4])<<8 | int(data[5])
	for i := 0; i < qdcount; i++ {
		// Skip name
		for offset < len(data) {
			length := int(data[offset])
			if length == 0 {
				offset++
				break
			}
			if length >= 0xC0 {
				// Pointer
				offset += 2
				break
			}
			offset += 1 + length
		}
		// Skip QTYPE and QCLASS
		offset += 4
	}

	// Parse answer section
	ancount := int(data[6])<<8 | int(data[7])
	resp := &Response{}

	for i := 0; i < ancount && offset < len(data); i++ {
		// Skip name (may be pointer)
		for offset < len(data) {
			length := int(data[offset])
			if length == 0 {
				offset++
				break
			}
			if length >= 0xC0 {
				offset += 2
				break
			}
			offset += 1 + length
		}

		if offset+10 > len(data) {
			break
		}

		// TYPE (2 bytes)
		rtype := uint16(data[offset])<<8 | uint16(data[offset+1])
		offset += 2

		// CLASS (2 bytes)
		offset += 2

		// TTL (4 bytes)
		ttl := uint32(data[offset])<<24 | uint32(data[offset+1])<<16 |
			uint32(data[offset+2])<<8 | uint32(data[offset+3])
		offset += 4

		// RDLENGTH (2 bytes)
		rdlen := int(data[offset])<<8 | int(data[offset+1])
		offset += 2

		if offset+rdlen > len(data) {
			break
		}

		// RDATA
		rdata := data[offset : offset+rdlen]
		offset += rdlen

		// For TXT records, strip length bytes
		if rtype == TypeTXT && len(rdata) > 0 {
			var txtData []byte
			pos := 0
			for pos < len(rdata) {
				length := int(rdata[pos])
				pos++
				if pos+length <= len(rdata) {
					txtData = append(txtData, rdata[pos:pos+length]...)
				}
				pos += length
			}
			rdata = txtData
		}

		resp.Records = append(resp.Records, rdata)
		if resp.TTL == 0 {
			resp.TTL = ttl
		}
	}

	// Combine all TXT records
	for _, r := range resp.Records {
		resp.Data = append(resp.Data, r...)
	}

	return resp, nil
}

// splitLabels splits a domain name into labels.
func splitLabels(name string) []string {
	var labels []string
	var current []byte

	for i := 0; i < len(name); i++ {
		if name[i] == '.' {
			if len(current) > 0 {
				labels = append(labels, string(current))
				current = nil
			}
		} else {
			current = append(current, name[i])
		}
	}
	if len(current) > 0 {
		labels = append(labels, string(current))
	}
	return labels
}
