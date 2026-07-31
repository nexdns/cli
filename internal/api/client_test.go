package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewClient(server.URL, "nxd_testtoken", 10*time.Second, false)
	return server, client
}

func TestClientAuthHeader(t *testing.T) {
	var gotAuth string
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(204)
	})

	_ = client.do(context.Background(), "GET", "/test", nil, nil)
	assert.Equal(t, "Bearer nxd_testtoken", gotAuth)
}

func TestClientUserAgent(t *testing.T) {
	var gotUA string
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(204)
	})

	_ = client.do(context.Background(), "GET", "/test", nil, nil)
	assert.Contains(t, gotUA, "nexdns-cli/")
}

func TestClientContentType(t *testing.T) {
	var gotCT string
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{}}`))
	})

	body := map[string]string{"key": "value"}
	var result map[string]interface{}
	_ = client.do(context.Background(), "POST", "/test", body, &result)
	assert.Equal(t, "application/json", gotCT)
}

func TestClientParsesEnvelope(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"email":"test@example.com","name":"Test"}}`))
	})

	var account Account
	err := client.do(context.Background(), "GET", "/account", nil, &account)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", account.Email)
	assert.Equal(t, "Test", account.Name)
}

func TestClientHandles204(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	err := client.do(context.Background(), "DELETE", "/zones/abc", nil, nil)
	assert.NoError(t, err)
}

func TestClientHandles400(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		w.Write([]byte(`{"status":"error","error":{"code":"validation_error","message":"Validation failed.","details":{"name":["Domain name is required."]}}}`))
	})

	var result interface{}
	err := client.do(context.Background(), "POST", "/zones", nil, &result)
	require.Error(t, err)

	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 400, apiErr.StatusCode)
	assert.Equal(t, "validation_error", apiErr.Code)
	assert.Contains(t, apiErr.Details["name"], "Domain name is required.")
}

func TestClientHandles404(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		w.Write([]byte(`{"status":"error","error":{"code":"not_found","message":"Zone not found."}}`))
	})

	var result interface{}
	err := client.do(context.Background(), "GET", "/zones/abc", nil, &result)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestClientHandles429(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Reset", "1700000000")
		w.WriteHeader(429)
		w.Write([]byte(`{"status":"error","error":{"code":"rate_limited","message":"Too many requests."}}`))
	})

	var result interface{}
	err := client.do(context.Background(), "GET", "/zones", nil, &result)
	require.Error(t, err)
	assert.True(t, IsRateLimited(err))
}

func TestClientNoRetryOn4xx(t *testing.T) {
	var attempts atomic.Int32
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		w.Write([]byte(`{"status":"error","error":{"code":"bad_request","message":"Bad request."}}`))
	})

	var result interface{}
	_ = client.do(context.Background(), "POST", "/zones", nil, &result)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestClientRetriesOn500(t *testing.T) {
	var attempts atomic.Int32
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			w.Write([]byte(`{"status":"error","error":{"code":"server_error","message":"Internal error."}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"email":"test@example.com"}}`))
	})

	var account Account
	err := client.do(context.Background(), "GET", "/account", nil, &account)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", account.Email)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestClientDoesNotRetryWritesOn500(t *testing.T) {
	// Non-idempotent writes must not be retried: a retry could double-submit and
	// surface a spurious 409/404 if the first attempt was actually applied.
	for _, method := range []string{"POST", "DELETE"} {
		t.Run(method, func(t *testing.T) {
			var attempts atomic.Int32
			_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(500)
				w.Write([]byte(`{"status":"error","error":{"code":"server_error","message":"Internal error."}}`))
			})

			err := client.do(context.Background(), method, "/zones", map[string]string{"name": "x"}, nil)
			require.Error(t, err)
			assert.Equal(t, int32(1), attempts.Load(), "%s must not be retried", method)
		})
	}
}

func TestClientParsesMetadata(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":[{"id":"abc","name":"example.com","type":"master","status":"active","ns_group":"Russia","records_count":5}],"meta":{"total":1,"page":1,"per_page":25,"last_page":1}}`))
	})

	var data []ZoneListItem
	var meta PaginationMeta
	err := client.doWithMeta(context.Background(), "GET", "/zones", nil, &data, &meta)
	require.NoError(t, err)
	assert.Len(t, data, 1)
	assert.Equal(t, "abc", data[0].ID)
	assert.Equal(t, "example.com", data[0].Name)
	assert.Equal(t, 1, meta.Total)
	assert.Equal(t, 1, meta.LastPage)
}

// A response that reports the budget spent must park the next request until the
// window rolls over, rather than spending a retry attempt on a certain 429. This
// is what keeps a bulk import from losing records once it outruns the plan.
func TestClientWaitsForTheRateLimitWindow(t *testing.T) {
	var starts []time.Time
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		starts = append(starts, time.Now())
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Limit", "180")
		if len(starts) == 1 {
			// Budget spent, window rolls over in a second.
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(2*time.Second).Unix(), 10))
		} else {
			w.Header().Set("X-RateLimit-Remaining", "179")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(60*time.Second).Unix(), 10))
		}
		w.Write([]byte(`{"status":"success","data":{}}`))
	})

	require.NoError(t, client.do(context.Background(), "GET", "/first", nil, nil))
	require.NoError(t, client.do(context.Background(), "GET", "/second", nil, nil))

	require.Len(t, starts, 2)
	gap := starts[1].Sub(starts[0])
	// The header carries whole seconds, so a reset 2s out is at least 1s away.
	assert.GreaterOrEqual(t, gap, 900*time.Millisecond, "the second request must wait for the window, waited %s", gap)
}

// A skewed or stale reset timestamp must not park a command for minutes.
func TestRateLimitWaitIsCapped(t *testing.T) {
	assert.Equal(t, time.Duration(0), parseRateLimitReset(""))
	assert.Equal(t, time.Duration(0), parseRateLimitReset("not-a-number"))
	assert.Equal(t, time.Duration(0), parseRateLimitReset(strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10)))
	// A far-off window is returned as-is; the caller decides not to wait for it.
	assert.Greater(t, parseRateLimitReset(strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)), maxRateLimitWait)
}

// The rate-limit error must keep the server's reason - which limit was hit - and
// add a readable wait, not replace the sentence with an epoch timestamp.
func TestRateLimitErrorKeepsTheServerReason(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(4*time.Minute).Unix(), 10))
		w.WriteHeader(429)
		w.Write([]byte(`{"status":"error","error":{"code":"rate_limit_exceeded","message":"Too many NS group changes for this zone."}}`))
	})

	err := client.do(context.Background(), "PATCH", "/zones/x", map[string]string{"ns_group": "eu"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Too many NS group changes for this zone.")
	assert.Contains(t, err.Error(), "Try again in 4m.")
	assert.NotContains(t, err.Error(), "Reset at")
}

func TestFormatWait(t *testing.T) {
	assert.Equal(t, "2s", formatWait(2*time.Second))
	assert.Equal(t, "45s", formatWait(45*time.Second))
	assert.Equal(t, "4m", formatWait(4*time.Minute))
	assert.Equal(t, "1h30m", formatWait(90*time.Minute))
}
