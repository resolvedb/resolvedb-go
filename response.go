package resolvedb

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Response represents a parsed ResolveDB response.
type Response struct {
	Version  string        // Protocol version (e.g., "rdb1")
	Status   string        // Status code (e.g., "ok", "notfound", "error")
	Type     string        // Response type (e.g., "json", "text", "binary")
	Encoding string        // Data encoding (e.g., "base64", "hex", "plain")
	Format   string        // Data format (e.g., "json", "text")
	TTL      time.Duration // Cache TTL
	Data     []byte        // Raw response data
	Error    string        // Error details if status != "ok"
	Chunks   int           // Number of chunks for large data
	ChunkID  int           // Current chunk ID
	Hash     string        // Content hash for verification
}

// ParseResponse parses a UQRP response string.
// Supports two formats:
// 1. JSON format: v=rdb1;s=<status>;t=<type>;d=<json_data>
// 2. Compact format: v=rdb1;s=ok;loc=Quebec;tc=-7.2;tf=19.0;...
func ParseResponse(s string) (*Response, error) {
	resp := &Response{}

	// Reserved keys that are not part of the data payload
	reservedKeys := map[string]bool{
		"v": true, "s": true, "t": true, "e": true, "f": true,
		"ttl": true, "d": true, "err": true, "chunks": true,
		"chunk": true, "hash": true, "ts": true,
	}

	// Collect non-reserved keys as data fields
	dataFields := make(map[string]any)

	parts := strings.Split(s, ";")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, value := kv[0], kv[1]

		switch key {
		case "v":
			resp.Version = value
		case "s":
			resp.Status = value
		case "t":
			resp.Type = value
		case "e":
			resp.Encoding = value
		case "f":
			resp.Format = value
		case "ttl":
			if ttl, err := strconv.Atoi(value); err == nil {
				resp.TTL = time.Duration(ttl) * time.Second
			}
		case "d":
			data, err := decodeResponseData(value, resp.Encoding)
			if err != nil {
				return nil, fmt.Errorf("decode data: %w", err)
			}
			resp.Data = data
		case "err":
			resp.Error = value
		case "chunks":
			if n, err := strconv.Atoi(value); err == nil {
				resp.Chunks = n
			}
		case "chunk":
			if n, err := strconv.Atoi(value); err == nil {
				resp.ChunkID = n
			}
		case "hash":
			resp.Hash = value
		case "ts":
			// Timestamp - reserved but not stored in Response
		default:
			// Non-reserved key - part of data payload
			if !reservedKeys[key] {
				dataFields[key] = parseValue(value)
			}
		}
	}

	// Validate required fields
	if resp.Version == "" {
		return nil, ErrInvalidResponse
	}

	// If no explicit d= field but we have data fields, convert to JSON
	if resp.Data == nil && len(dataFields) > 0 {
		// Expand compact field names to full names for weather data
		expanded := expandCompactFields(dataFields)
		jsonData, err := json.Marshal(expanded)
		if err != nil {
			return nil, fmt.Errorf("marshal data fields: %w", err)
		}
		resp.Data = jsonData
	}

	return resp, nil
}

// parseValue attempts to parse a string value as a number if possible.
func parseValue(s string) any {
	// Try integer
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	// Try float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	// Try boolean
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	// Return as string
	return s
}

// expandCompactFields expands compact UQRP field names to full JSON field names.
func expandCompactFields(fields map[string]any) map[string]any {
	// Mapping of compact names to full names
	fieldMap := map[string]string{
		"loc": "location",
		"tc":  "temp_c",
		"tf":  "temp_f",
		"cnd": "conditions",
		"hum": "humidity",
		"wnd": "wind_kph",
		"vis": "visibility_km",
		"uv":  "uv_index",
		"tz":  "timezone",
		"lt":  "local_time",
		// GeoIP fields
		"ip":      "ip",
		"cc":      "country_code",
		"cn":      "country",
		"rg":      "region",
		"ct":      "city",
		"lat":     "latitude",
		"lon":     "longitude",
		"isp":     "isp",
		"org":     "organization",
		"as":      "asn",
		"mobile":  "mobile",
		"proxy":   "proxy",
		"hosting": "hosting",
	}

	expanded := make(map[string]any)
	for k, v := range fields {
		if fullName, ok := fieldMap[k]; ok {
			expanded[fullName] = v
		} else {
			expanded[k] = v
		}
	}
	return expanded
}

// decodeResponseData decodes the data field based on encoding.
func decodeResponseData(data, encoding string) ([]byte, error) {
	switch encoding {
	case "base64", "b64":
		return decodeBase64(data)
	case "hex":
		return decodeHex(data)
	case "plain", "text", "":
		return []byte(data), nil
	default:
		// Try base64 as default for unknown encodings
		if decoded, err := decodeBase64(data); err == nil {
			return decoded, nil
		}
		return []byte(data), nil
	}
}

// IsSuccess returns true if the response indicates success.
func (r *Response) IsSuccess() bool {
	return r.Status == "ok" || r.Status == "success"
}

// IsError returns true if the response indicates an error.
func (r *Response) IsError() bool {
	return r.Status == "error" || strings.HasPrefix(r.Status, "E0")
}

// Unmarshal decodes the response data into v.
func (r *Response) Unmarshal(v any) error {
	if r.Data == nil {
		return ErrNotFound
	}

	switch r.Format {
	case "json", "":
		if err := json.Unmarshal(r.Data, v); err != nil {
			return fmt.Errorf("json unmarshal: %w", err)
		}
		return nil
	case "text":
		if s, ok := v.(*string); ok {
			*s = string(r.Data)
			return nil
		}
		return fmt.Errorf("cannot unmarshal text into %T", v)
	default:
		// Try JSON first
		if err := json.Unmarshal(r.Data, v); err == nil {
			return nil
		}
		return fmt.Errorf("unsupported format: %s", r.Format)
	}
}

// String returns the raw data as a string.
func (r *Response) String() string {
	return string(r.Data)
}

// ToError converts the response to an error if it indicates failure.
func (r *Response) ToError() error {
	if r.IsSuccess() {
		return nil
	}

	// Check if status is an error code
	if strings.HasPrefix(r.Status, "E0") {
		return errorFromCode(r.Status, r.Error)
	}

	// Map status strings to error codes
	switch r.Status {
	case "notfound":
		return errorFromCode(CodeNotFound, r.Error)
	case "unauthorized":
		return errorFromCode(CodeUnauthorized, r.Error)
	case "forbidden":
		return errorFromCode(CodeForbidden, r.Error)
	case "ratelimit", "ratelimited":
		return errorFromCode(CodeRateLimited, r.Error)
	case "timeout":
		return errorFromCode(CodeTimeout, r.Error)
	case "error":
		if r.Error != "" && strings.HasPrefix(r.Error, "E0") {
			code := r.Error[:4]
			details := ""
			if len(r.Error) > 5 {
				details = r.Error[5:]
			}
			return errorFromCode(code, details)
		}
		return errorFromCode(CodeServerError, r.Error)
	default:
		return &Error{Code: CodeServerError, Message: r.Status, Details: r.Error}
	}
}

// IsChunked returns true if the response is part of a chunked data set.
func (r *Response) IsChunked() bool {
	return r.Chunks > 1
}
