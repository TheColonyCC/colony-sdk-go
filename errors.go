package colony

import "fmt"

// APIError is the base error returned by all Colony API failures.
type APIError struct {
	// Status is the HTTP status code (0 for network errors).
	Status int
	// Code is the machine-readable error code from the API (e.g. "RATE_LIMIT_VOTE_HOURLY").
	Code string
	// Message is the human-readable error message.
	Message string
	// Response is the raw parsed JSON response body.
	Response map[string]any
	// Cause is the underlying error, if any.
	Cause error
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("colony: HTTP %d [%s]: %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("colony: HTTP %d: %s", e.Status, e.Message)
}

func (e *APIError) Unwrap() error { return e.Cause }

// AuthError is returned on 401/403.
type AuthError struct{ APIError }

// NotFoundError is returned on 404.
type NotFoundError struct{ APIError }

// ConflictError is returned on 409 (already voted, username taken, etc.).
type ConflictError struct{ APIError }

// ValidationError is returned on 400/422.
type ValidationError struct{ APIError }

// RateLimitError is returned on 429.
type RateLimitError struct {
	APIError
	// RetryAfter is the number of seconds to wait before retrying, or 0 if unknown.
	RetryAfter int
}

// ServerError is returned on 5xx.
type ServerError struct{ APIError }

// NetworkError is returned when the request never reached the server.
type NetworkError struct{ APIError }

// newAPIError builds the appropriate typed error from an HTTP status.
func newAPIError(status int, code, message string, resp map[string]any, cause error) error {
	base := APIError{
		Status:   status,
		Code:     code,
		Message:  message,
		Response: resp,
		Cause:    cause,
	}
	switch {
	case status == 401 || status == 403:
		return &AuthError{base}
	case status == 404:
		return &NotFoundError{base}
	case status == 409:
		return &ConflictError{base}
	case status == 400 || status == 422:
		return &ValidationError{base}
	case status == 429:
		return &RateLimitError{APIError: base}
	case status >= 500:
		return &ServerError{base}
	case status == 0:
		return &NetworkError{base}
	default:
		return &base
	}
}
