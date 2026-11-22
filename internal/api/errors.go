package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// APIError represents an error returned by the NexDNS API.
type APIError struct {
	StatusCode int
	Code       string          `json:"code"`
	Message    string          `json:"message"`
	Details    ValidationField `json:"details,omitempty"`
}

// ValidationField holds the messages for each field. The API sends either a
// list of messages or a single string.
type ValidationField map[string][]string

func (d *ValidationField) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	out := make(ValidationField, len(raw))
	for field, value := range raw {
		var list []string
		if err := json.Unmarshal(value, &list); err == nil {
			out[field] = list
			continue
		}

		var single string
		if err := json.Unmarshal(value, &single); err == nil {
			out[field] = []string{single}
			continue
		}
		// Any other JSON value is kept as raw text.
		out[field] = []string{string(value)}
	}

	*d = out
	return nil
}

func (e *APIError) Error() string {
	if len(e.Details) > 0 {
		var parts []string
		for field, msgs := range e.Details {
			for _, msg := range msgs {
				parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
			}
		}
		return fmt.Sprintf("%s (%s)", e.Message, strings.Join(parts, "; "))
	}
	return e.Message
}

// parseErrorResponse parses an API error response body.
// Handles both formats: {"status":"error","error":{...}} and {"error":{...}}
func parseErrorResponse(statusCode int, body []byte) *APIError {
	// Try format with status wrapper
	var withStatus struct {
		Error *APIError `json:"error"`
	}
	if err := json.Unmarshal(body, &withStatus); err == nil && withStatus.Error != nil {
		withStatus.Error.StatusCode = statusCode
		if withStatus.Error.Message == "" {
			withStatus.Error.Message = fmt.Sprintf("API error (HTTP %d)", statusCode)
		}
		return withStatus.Error
	}

	// Fallback to generic error
	return &APIError{
		StatusCode: statusCode,
		Message:    fmt.Sprintf("API error (HTTP %d)", statusCode),
	}
}

// IsNotFound returns true if the error is a 404 Not Found.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 404
}

// IsConflict returns true if the error is a 409 Conflict.
func IsConflict(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 409
}

// IsRateLimited returns true if the error is a 429 Too Many Requests.
func IsRateLimited(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 429
}
