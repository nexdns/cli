package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nexdns/cli/internal/version"
)

const (
	maxRetries = 3
	baseDelay  = 1 * time.Second
	maxJitter  = 500 * time.Millisecond
	// maxRateLimitWait is the longest a command will sit waiting for a rate-limit
	// window to roll over. Beyond it the limit is reported instead: a per-minute
	// request budget is worth waiting out, while some limits reset only after a
	// day, and a command that hangs for hours is worse than one that says which
	// limit it hit.
	maxRateLimitWait = 90 * time.Second
)

// Client is the NexDNS API client.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	Verbose    bool
	UserAgent  string

	// budget tracks the rate-limit headers of the last response so the next
	// request can wait for the window to roll over instead of spending an attempt
	// on a 429. Bulk work - a zone import over a few hundred rrsets - otherwise
	// outruns the plan's per-minute budget and starts losing individual records.
	budgetMu        sync.Mutex
	budgetRemaining int
	budgetReset     time.Time
	budgetKnown     bool
}

// NewClient creates a new API client.
func NewClient(baseURL, token string, timeout time.Duration, verbose bool) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		Verbose:   verbose,
		UserAgent: fmt.Sprintf("nexdns-cli/%s", version.Version),
	}
}

// apiResponse is the envelope for successful API responses.
type apiResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Meta   json.RawMessage `json:"meta,omitempty"`
}

// do executes an HTTP request with retry logic.
func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	return c.doWithMeta(ctx, method, path, body, result, nil)
}

// doWithMeta executes an HTTP request and optionally extracts pagination meta.
func (c *Client) doWithMeta(ctx context.Context, method, path string, body interface{}, result interface{}, meta interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.BaseURL + path

	// Only GET is safe to retry automatically. POST is not idempotent; PUT/DELETE can
	// surface spurious 409/404 on a slow backend because the first attempt may have already
	// been applied (and record IDs are content-derived, so they change on update). Retrying
	// a write that timed out would double-submit and report "already exists" / "not found".
	retryable := method == http.MethodGet

	c.waitForBudget()

	var lastErr error
	for attempt := range maxRetries {
		// Re-create body reader for retries
		if attempt > 0 && body != nil {
			data, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(data)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.UserAgent)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		if c.Verbose {
			fmt.Printf("→ %s %s\n", method, url)
		}

		start := time.Now()
		resp, err := c.HTTPClient.Do(req)
		duration := time.Since(start)

		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if retryable && attempt < maxRetries-1 {
				if c.Verbose {
					fmt.Printf("  ✗ %v (retry %d/%d)\n", err, attempt+1, maxRetries-1)
				}
				sleep(attempt)
				continue
			}
			return lastErr
		}

		c.recordBudget(resp.Header)

		if c.Verbose {
			fmt.Printf("← %d %s (%s)\n", resp.StatusCode, resp.Status, duration.Round(time.Millisecond))
			// Log rate limit headers
			if rl := resp.Header.Get("X-RateLimit-Remaining"); rl != "" {
				fmt.Printf("  Rate limit: %s/%s remaining (reset: %s)\n",
					rl,
					resp.Header.Get("X-RateLimit-Limit"),
					resp.Header.Get("X-RateLimit-Reset"))
			}
		}

		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		// 204 No Content
		if resp.StatusCode == http.StatusNoContent {
			return nil
		}

		// 5xx – retry (idempotent reads only)
		if retryable && resp.StatusCode >= 500 && attempt < maxRetries-1 {
			lastErr = parseErrorResponse(resp.StatusCode, respBody)
			if c.Verbose {
				fmt.Printf("  ✗ server error %d (retry %d/%d)\n", resp.StatusCode, attempt+1, maxRetries-1)
			}
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			if retryAfter > 0 {
				time.Sleep(retryAfter)
			} else {
				sleep(attempt)
			}
			continue
		}

		// 429 - rate limited. Retried for every method, unlike a 5xx: the budget
		// is checked before the request is executed, so a rejected write had no
		// effect and replaying it cannot double-apply anything. A bulk operation
		// that outruns the per-minute budget therefore waits rather than losing
		// the operations the budget rejected.
		if resp.StatusCode == http.StatusTooManyRequests {
			apiErr := parseErrorResponse(resp.StatusCode, respBody)
			wait := parseRetryAfter(resp.Header.Get("Retry-After"))
			if wait <= 0 {
				wait = parseRateLimitReset(resp.Header.Get("X-RateLimit-Reset"))
			}

			// Keep the server's reason and add how long the window has left. The
			// message used to be replaced with "Rate limited. Reset at 1785370484",
			// which threw away the one sentence that said *which* limit was hit
			// ("Too many NS group changes for this zone.") in favour of an epoch
			// timestamp no one can read.
			if wait > 0 {
				apiErr.Message = strings.TrimSpace(apiErr.Message)
				if apiErr.Message == "" {
					apiErr.Message = "Rate limited."
				}
				apiErr.Message = fmt.Sprintf("%s Try again in %s.", apiErr.Message, formatWait(wait))
			}

			// A window too far out to wait for is reported, not slept through.
			if wait <= maxRateLimitWait && attempt < maxRetries-1 {
				lastErr = apiErr
				if c.Verbose {
					fmt.Printf("  ✗ rate limited, waiting %s (retry %d/%d)\n", wait.Round(time.Second), attempt+1, maxRetries-1)
				}
				if wait > 0 {
					time.Sleep(wait)
				} else {
					sleep(attempt)
				}
				continue
			}

			return apiErr
		}

		// 4xx/5xx errors
		if resp.StatusCode >= 400 {
			return parseErrorResponse(resp.StatusCode, respBody)
		}

		// Parse success response
		if result == nil {
			return nil
		}

		// Try to parse as envelope {"status":"success","data":...}
		var envelope apiResponse
		if err := json.Unmarshal(respBody, &envelope); err == nil && envelope.Status == "success" {
			if result != nil && len(envelope.Data) > 0 {
				if err := json.Unmarshal(envelope.Data, result); err != nil {
					return fmt.Errorf("parsing response data: %w", err)
				}
			}
			if meta != nil && len(envelope.Meta) > 0 {
				if err := json.Unmarshal(envelope.Meta, meta); err != nil {
					return fmt.Errorf("parsing response meta: %w", err)
				}
			}
			return nil
		}

		// Fallback: parse body directly (e.g., for export endpoint)
		if err := json.Unmarshal(respBody, result); err != nil {
			// A body that is not JSON at all (a plain-text zone export) is
			// handed over verbatim to a caller that can hold it as-is.
			if s, ok := result.(*string); ok {
				*s = string(respBody)
				return nil
			}
			if raw, ok := result.(*json.RawMessage); ok {
				*raw = json.RawMessage(strconv.Quote(string(respBody)))
				return nil
			}
			return fmt.Errorf("parsing response: %w", err)
		}

		return nil
	}

	return lastErr
}

func sleep(attempt int) {
	delay := baseDelay * time.Duration(math.Pow(2, float64(attempt)))
	jitter := time.Duration(rand.Int64N(int64(maxJitter)))
	time.Sleep(delay + jitter)
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

// recordBudget stores the rate-limit headers of a response.
func (c *Client) recordBudget(h http.Header) {
	remaining := h.Get("X-RateLimit-Remaining")
	if remaining == "" {
		return
	}

	value, err := strconv.Atoi(remaining)
	if err != nil {
		return
	}

	reset, err := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64)
	if err != nil {
		return
	}

	c.budgetMu.Lock()
	defer c.budgetMu.Unlock()
	c.budgetRemaining = value
	c.budgetReset = time.Unix(reset, 0)
	c.budgetKnown = true
}

// waitForBudget sleeps until the rate-limit window rolls over when the previous
// response reported the budget exhausted. This is the difference between a bulk
// operation taking a little longer and a bulk operation losing records: a 429 is
// retried, but only a few times, so a long enough overrun still ran out of
// attempts.
func (c *Client) waitForBudget() {
	c.budgetMu.Lock()
	known, remaining, reset := c.budgetKnown, c.budgetRemaining, c.budgetReset
	c.budgetMu.Unlock()

	if !known || remaining > 0 {
		return
	}

	wait := time.Until(reset)
	if wait <= 0 || wait > maxRateLimitWait {
		// Nothing to wait for, or a window so far out that the request should go
		// and come back with the limit rather than the command hanging.
		return
	}

	if c.Verbose {
		fmt.Printf("  … rate-limit budget spent, waiting %s for the window to reset\n", wait.Round(time.Second))
	}
	time.Sleep(wait)
}

// formatWait renders a retry delay the way a person reads it: seconds for a
// per-minute budget, minutes or hours for the longer per-operation limits.
func formatWait(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Round(time.Second).Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Round(time.Minute).Minutes()))
	default:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Round(time.Minute).Minutes())%60)
	}
}

// parseRateLimitReset turns an X-RateLimit-Reset unix timestamp into how long is
// left of the window. Used when the server sends no Retry-After. The value is
// returned as-is - the caller decides whether it is short enough to wait for,
// because that answer differs between a per-minute request budget and a per-day
// operation limit.
func parseRateLimitReset(value string) time.Duration {
	if value == "" {
		return 0
	}

	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}

	wait := time.Until(time.Unix(seconds, 0))
	if wait <= 0 {
		return 0
	}

	return wait
}

// buildQuery builds a query string from key-value pairs, skipping empty values.
func buildQuery(params map[string]string) string {
	var parts []string
	for k, v := range params {
		if v != "" {
			parts = append(parts, k+"="+v)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "?" + strings.Join(parts, "&")
}
