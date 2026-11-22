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
	// maxRateLimitWait is the longest a command waits for a rate-limit window
	// to reset. A longer wait is reported as an error instead.
	maxRateLimitWait = 90 * time.Second
)

// Client is the NexDNS API client.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
	Verbose    bool
	UserAgent  string

	// budget holds the rate-limit headers of the last response, so the next
	// request can wait for the window to reset instead of getting a 429.
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

	// Only GET is retried on a network error or a 5xx: a write that failed may
	// already have been applied.
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

		if resp.StatusCode == http.StatusNoContent {
			return nil
		}

		// 5xx: retry idempotent reads only
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

		// 429: retry any method. The budget is checked before the request runs,
		// so a rejected write had no effect.
		if resp.StatusCode == http.StatusTooManyRequests {
			apiErr := parseErrorResponse(resp.StatusCode, respBody)
			wait := parseRetryAfter(resp.Header.Get("Retry-After"))
			if wait <= 0 {
				wait = parseRateLimitReset(resp.Header.Get("X-RateLimit-Reset"))
			}

			// Keep the server's message, which names the limit, and add the wait.
			if wait > 0 {
				apiErr.Message = strings.TrimSpace(apiErr.Message)
				if apiErr.Message == "" {
					apiErr.Message = "Rate limited."
				}
				apiErr.Message = fmt.Sprintf("%s Try again in %s.", apiErr.Message, formatWait(wait))
			}

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

		if resp.StatusCode >= 400 {
			return parseErrorResponse(resp.StatusCode, respBody)
		}

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
			// Plain text, such as a BIND zone export.
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

// waitForBudget sleeps until the rate-limit window resets when the previous
// response reported the budget exhausted.
func (c *Client) waitForBudget() {
	c.budgetMu.Lock()
	known, remaining, reset := c.budgetKnown, c.budgetRemaining, c.budgetReset
	c.budgetMu.Unlock()

	if !known || remaining > 0 {
		return
	}

	wait := time.Until(reset)
	if wait <= 0 || wait > maxRateLimitWait {
		// Nothing to wait for, or too long to wait: the server reports the limit.
		return
	}

	if c.Verbose {
		fmt.Printf("  rate-limit budget spent, waiting %s for the window to reset\n", wait.Round(time.Second))
	}
	time.Sleep(wait)
}

// formatWait renders a wait in seconds, minutes or hours.
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

// parseRateLimitReset converts an X-RateLimit-Reset unix timestamp into the
// time left in the window. It is used when there is no Retry-After header.
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
