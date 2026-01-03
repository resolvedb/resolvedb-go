package resolvedb

import (
	"errors"
	"fmt"
)

// Standard error codes from ResolveDB protocol.
const (
	CodeSuccess        = "E000" // Success
	CodeBadRequest     = "E001" // Malformed query
	CodeUnauthorized   = "E002" // Missing or invalid auth
	CodeForbidden      = "E003" // Insufficient permissions
	CodeNotFound       = "E004" // Resource not found
	CodeConflict       = "E005" // Resource already exists
	CodePayloadTooLarge = "E006" // Data exceeds limits
	CodeInvalidFormat  = "E007" // Invalid data format
	CodeVersionMismatch = "E008" // Version conflict
	CodeNamespaceError = "E009" // Namespace issues
	CodeServerError    = "E010" // Internal error (retryable)
	CodeUnavailable    = "E011" // Service unavailable
	CodeTimeout        = "E012" // Query timeout (retryable)
	CodeRateLimited    = "E013" // Rate limit exceeded (retryable)
	CodeEncryptionRequired = "E014" // Encryption required
)

// Sentinel errors for use with errors.Is.
var (
	ErrBadRequest          = &Error{Code: CodeBadRequest, Message: "malformed query"}
	ErrUnauthorized        = &Error{Code: CodeUnauthorized, Message: "authentication required"}
	ErrForbidden           = &Error{Code: CodeForbidden, Message: "insufficient permissions"}
	ErrNotFound            = &Error{Code: CodeNotFound, Message: "resource not found"}
	ErrConflict            = &Error{Code: CodeConflict, Message: "resource already exists"}
	ErrPayloadTooLarge     = &Error{Code: CodePayloadTooLarge, Message: "data exceeds size limit"}
	ErrInvalidFormat       = &Error{Code: CodeInvalidFormat, Message: "invalid data format"}
	ErrVersionMismatch     = &Error{Code: CodeVersionMismatch, Message: "version conflict"}
	ErrNamespaceError      = &Error{Code: CodeNamespaceError, Message: "namespace error"}
	ErrServerError         = &Error{Code: CodeServerError, Message: "internal server error"}
	ErrUnavailable         = &Error{Code: CodeUnavailable, Message: "service unavailable"}
	ErrTimeout             = &Error{Code: CodeTimeout, Message: "query timeout"}
	ErrRateLimited         = &Error{Code: CodeRateLimited, Message: "rate limit exceeded"}
	ErrEncryptionRequired  = &Error{Code: CodeEncryptionRequired, Message: "encryption required"}

	// SDK-specific errors.
	ErrNonceExhausted           = errors.New("resolvedb: nonce counter exhausted, rotate encryption key")
	ErrEncryptedTransportRequired = errors.New("resolvedb: authenticated requests require encrypted transport")
	ErrInvalidResponse          = errors.New("resolvedb: invalid response format")
	ErrChunkIntegrity           = errors.New("resolvedb: chunk integrity verification failed")
	ErrForbiddenAlgorithm       = errors.New("resolvedb: forbidden JWT algorithm")
)

// Error represents a ResolveDB protocol error.
type Error struct {
	Code    string // Error code (E001-E014)
	Message string // Human-readable message
	Details string // Additional details from server
}

func (e *Error) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("resolvedb [%s]: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("resolvedb [%s]: %s", e.Code, e.Message)
}

// Is implements errors.Is for error comparison.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Retryable returns true if the error is transient and the request can be retried.
func (e *Error) Retryable() bool {
	switch e.Code {
	case CodeServerError, CodeTimeout, CodeRateLimited:
		return true
	default:
		return false
	}
}

// IsRetryable checks if an error is retryable.
func IsRetryable(err error) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Retryable()
	}
	return false
}

// IsNotFound checks if an error indicates a resource was not found.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized checks if an error indicates authentication is required.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsRateLimited checks if an error indicates rate limiting.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

// errorFromCode creates an Error from a protocol error code.
func errorFromCode(code, details string) error {
	switch code {
	case CodeSuccess:
		return nil
	case CodeBadRequest:
		return &Error{Code: code, Message: "malformed query", Details: details}
	case CodeUnauthorized:
		return &Error{Code: code, Message: "authentication required", Details: details}
	case CodeForbidden:
		return &Error{Code: code, Message: "insufficient permissions", Details: details}
	case CodeNotFound:
		return &Error{Code: code, Message: "resource not found", Details: details}
	case CodeConflict:
		return &Error{Code: code, Message: "resource already exists", Details: details}
	case CodePayloadTooLarge:
		return &Error{Code: code, Message: "data exceeds size limit", Details: details}
	case CodeInvalidFormat:
		return &Error{Code: code, Message: "invalid data format", Details: details}
	case CodeVersionMismatch:
		return &Error{Code: code, Message: "version conflict", Details: details}
	case CodeNamespaceError:
		return &Error{Code: code, Message: "namespace error", Details: details}
	case CodeServerError:
		return &Error{Code: code, Message: "internal server error", Details: details}
	case CodeUnavailable:
		return &Error{Code: code, Message: "service unavailable", Details: details}
	case CodeTimeout:
		return &Error{Code: code, Message: "query timeout", Details: details}
	case CodeRateLimited:
		return &Error{Code: code, Message: "rate limit exceeded", Details: details}
	case CodeEncryptionRequired:
		return &Error{Code: code, Message: "encryption required", Details: details}
	default:
		return &Error{Code: code, Message: "unknown error", Details: details}
	}
}
