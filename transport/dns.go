package transport

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"
)

// DNS implements traditional DNS transport over UDP/TCP.
type DNS struct {
	servers []string
	timeout time.Duration
}

// DNSOption configures a DNS transport.
type DNSOption func(*DNS)

// WithDNSServers sets the DNS servers to use.
func WithDNSServers(servers ...string) DNSOption {
	return func(d *DNS) {
		d.servers = servers
	}
}

// WithDNSTimeout sets the query timeout.
func WithDNSTimeout(timeout time.Duration) DNSOption {
	return func(d *DNS) {
		d.timeout = timeout
	}
}

// NewDNS creates a new traditional DNS transport.
func NewDNS(opts ...DNSOption) *DNS {
	d := &DNS{
		servers: []string{"8.8.8.8:53", "8.8.4.4:53"},
		timeout: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *DNS) Name() string { return "dns" }

// IsEncrypted returns false - traditional DNS is not encrypted.
// SECURITY WARNING: Do not use this transport for authenticated requests.
func (d *DNS) IsEncrypted() bool { return false }

func (d *DNS) Close() error { return nil }

// Query sends a DNS query over UDP.
func (d *DNS) Query(ctx context.Context, req *Request) (*Response, error) {
	wireMsg := buildDNSQuery(req.Name, req.Type)

	var lastErr error
	for _, server := range d.servers {
		resp, err := d.queryServer(ctx, server, wireMsg)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (d *DNS) queryServer(ctx context.Context, server string, query []byte) (*Response, error) {
	// Create UDP connection
	dialer := net.Dialer{Timeout: d.timeout}
	conn, err := dialer.DialContext(ctx, "udp", server)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", server, err)
	}
	defer conn.Close()

	// Set deadline
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(d.timeout)
	}
	conn.SetDeadline(deadline)

	// Send query
	if _, err := conn.Write(query); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	// Read response
	buf := make([]byte, 65535)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	return parseDNSResponse(buf[:n])
}

// QueryTCP sends a DNS query over TCP (for large responses).
func (d *DNS) QueryTCP(ctx context.Context, req *Request) (*Response, error) {
	wireMsg := buildDNSQuery(req.Name, req.Type)

	// Prepend 2-byte length for TCP
	tcpMsg := make([]byte, len(wireMsg)+2)
	tcpMsg[0] = byte(len(wireMsg) >> 8)
	tcpMsg[1] = byte(len(wireMsg) & 0xFF)
	copy(tcpMsg[2:], wireMsg)

	var lastErr error
	for _, server := range d.servers {
		resp, err := d.queryServerTCP(ctx, server, tcpMsg)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (d *DNS) queryServerTCP(ctx context.Context, server string, query []byte) (*Response, error) {
	dialer := net.Dialer{Timeout: d.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", server)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", server, err)
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(d.timeout)
	}
	conn.SetDeadline(deadline)

	if _, err := conn.Write(query); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	// Read length - use io.ReadFull to ensure complete read
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, fmt.Errorf("read length: %w", err)
	}
	length := int(lenBuf[0])<<8 | int(lenBuf[1])

	// Limit response size (64KB max per security review)
	if length > 65535 {
		return nil, fmt.Errorf("response too large: %d bytes", length)
	}

	// Read response - use io.ReadFull to ensure complete read
	buf := make([]byte, length)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	return parseDNSResponse(buf)
}
