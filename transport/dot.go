package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"
)

// DoT implements DNS-over-TLS transport.
type DoT struct {
	servers   []string
	timeout   time.Duration
	tlsConfig *tls.Config
}

// DoTOption configures a DoT transport.
type DoTOption func(*DoT)

// WithDoTServers sets the DoT servers to use.
func WithDoTServers(servers ...string) DoTOption {
	return func(d *DoT) {
		d.servers = servers
	}
}

// WithDoTTimeout sets the query timeout.
func WithDoTTimeout(timeout time.Duration) DoTOption {
	return func(d *DoT) {
		d.timeout = timeout
	}
}

// WithDoTTLSConfig sets custom TLS configuration.
func WithDoTTLSConfig(config *tls.Config) DoTOption {
	return func(d *DoT) {
		d.tlsConfig = config
	}
}

// NewDoT creates a new DNS-over-TLS transport.
func NewDoT(opts ...DoTOption) *DoT {
	d := &DoT{
		servers: []string{"1.1.1.1:853", "8.8.8.8:853"},
		timeout: 10 * time.Second,
		tlsConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *DoT) Name() string { return "dot" }

func (d *DoT) IsEncrypted() bool { return true }

func (d *DoT) Close() error { return nil }

// Query sends a DNS query over TLS.
func (d *DoT) Query(ctx context.Context, req *Request) (*Response, error) {
	wireMsg := buildDNSQuery(req.Name, req.Type)

	// Prepend 2-byte length for TCP
	tcpMsg := make([]byte, len(wireMsg)+2)
	tcpMsg[0] = byte(len(wireMsg) >> 8)
	tcpMsg[1] = byte(len(wireMsg) & 0xFF)
	copy(tcpMsg[2:], wireMsg)

	var lastErr error
	for _, server := range d.servers {
		resp, err := d.queryServer(ctx, server, tcpMsg)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (d *DoT) queryServer(ctx context.Context, server string, query []byte) (*Response, error) {
	// Parse server address
	host, _, err := net.SplitHostPort(server)
	if err != nil {
		host = server
	}

	// Create TLS config with server name
	tlsConfig := d.tlsConfig.Clone()
	if tlsConfig.ServerName == "" {
		tlsConfig.ServerName = host
	}

	// Dial with TLS
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{Timeout: d.timeout},
		Config:    tlsConfig,
	}

	conn, err := dialer.DialContext(ctx, "tcp", server)
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
