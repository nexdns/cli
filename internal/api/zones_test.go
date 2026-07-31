package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListZones(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/zones", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":[{"id":"xK9mQ2","name":"example.com","type":"master","status":"active","ns_group":"Russia","records_count":12,"created_at":"2025-01-15"}],"meta":{"total":1,"page":1,"per_page":25,"last_page":1}}`))
	})

	resp, err := client.ListZones(context.Background(), ListZonesOptions{})
	require.NoError(t, err)
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, "example.com", resp.Data[0].Name)
	assert.Equal(t, "xK9mQ2", resp.Data[0].ID)
	assert.Equal(t, 12, resp.Data[0].RecordsCount)
}

func TestGetZoneByNameExactMatch(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Search returns partial matches too
		w.Write([]byte(`{"status":"success","data":[{"id":"aaa","name":"sub.example.com","type":"master","status":"active","ns_group":"Russia"},{"id":"bbb","name":"example.com","type":"master","status":"active","ns_group":"Russia"}],"meta":{"total":2,"page":1,"per_page":100,"last_page":1}}`))
	})

	zone, err := client.GetZoneByName(context.Background(), "example.com")
	require.NoError(t, err)
	assert.Equal(t, "bbb", zone.ID)
	assert.Equal(t, "example.com", zone.Name)
}

func TestGetZoneByNameNotFound(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":[],"meta":{"total":0,"page":1,"per_page":100,"last_page":1}}`))
	})

	_, err := client.GetZoneByName(context.Background(), "notexist.com")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestCreateZone(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var req CreateZoneRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "example.com", req.Name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write([]byte(`{"status":"success","data":{"id":"newzone","name":"example.com","type":"master","status":"active","ns_group":"Russia"}}`))
	})

	zone, err := client.CreateZone(context.Background(), CreateZoneRequest{Name: "example.com"})
	require.NoError(t, err)
	assert.Equal(t, "newzone", zone.ID)
}

func TestCreateZoneConflict(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(409)
		w.Write([]byte(`{"status":"error","error":{"code":"conflict","message":"Zone already exists."}}`))
	})

	_, err := client.CreateZone(context.Background(), CreateZoneRequest{Name: "example.com"})
	require.Error(t, err)
	assert.True(t, IsConflict(err))
}

func TestDeleteZone(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/zones/xK9mQ2", r.URL.Path)
		w.WriteHeader(204)
	})

	err := client.DeleteZone(context.Background(), "xK9mQ2")
	assert.NoError(t, err)
}

func TestExportZone(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "bind", r.URL.Query().Get("format"))
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(`; Zone: example.com
example.com. 3600 IN A 192.168.1.1`))
	})

	result, err := client.ExportZone(context.Background(), "xK9mQ2", "bind")
	require.NoError(t, err)
	assert.Contains(t, result, "example.com")
}

func TestMoveZoneSendsTheSlug(t *testing.T) {
	var body map[string]any
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		assert.Equal(t, "/zones/xK9mQ2", r.URL.Path)
		raw, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(raw, &body))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"id":"xK9mQ2","name":"example.com","type":"master","status":"active","ns_group":{"id":"eu","slug":"eu","name":"Europe"},"records_count":12,"created_at":"2026-01-15","updated_at":"2026-07-30"}}`))
	})

	zone, err := client.MoveZone(context.Background(), "xK9mQ2", "eu")
	require.NoError(t, err)
	require.NotNil(t, zone.NSGroup)
	assert.Equal(t, "eu", zone.NSGroup.Slug)
	assert.Equal(t, "eu", body["ns_group"])
}

// A 429 is checked before the request is executed, so replaying a rejected write
// cannot double-apply it - the retry covers writes too, which is what lets a
// large import outrun a per-minute budget instead of losing records.
func TestMoveZoneRetriesARateLimitAndSucceeds(t *testing.T) {
	calls := 0
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(429)
			w.Write([]byte(`{"status":"error","error":{"code":"rate_limit_exceeded","message":"Too many requests."}}`))
			return
		}
		w.Write([]byte(`{"status":"success","data":{"id":"xK9mQ2","name":"example.com","type":"master","status":"active","ns_group":{"id":"eu","slug":"eu","name":"Europe"},"records_count":12,"created_at":"2026-01-15","updated_at":"2026-07-30"}}`))
	})

	zone, err := client.MoveZone(context.Background(), "xK9mQ2", "eu")
	require.NoError(t, err)
	require.NotNil(t, zone.NSGroup)
	assert.Equal(t, 2, calls, "the rejected attempt must be replayed once")
}

// A 5xx on a write is a different story: the request may have been executed, so
// it is never replayed.
func TestMoveZoneDoesNotRetryAServerError(t *testing.T) {
	calls := 0
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{"status":"error","error":{"code":"server_error","message":"An internal error occurred"}}`))
	})

	_, err := client.MoveZone(context.Background(), "xK9mQ2", "eu")
	require.Error(t, err)
	assert.Equal(t, 1, calls, "a PATCH must not be replayed after a server error")
}

// The rate-limit retry must give up rather than loop forever.
func TestRateLimitRetryEventuallySurfaces(t *testing.T) {
	calls := 0
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(429)
		w.Write([]byte(`{"status":"error","error":{"code":"rate_limit_exceeded","message":"Too many requests."}}`))
	})

	_, err := client.MoveZone(context.Background(), "xK9mQ2", "eu")
	require.Error(t, err)
	assert.True(t, IsRateLimited(err))
	assert.Equal(t, maxRetries, calls)
}

// A per-day limit (an NS-group move is 3 a day per zone) must be reported at
// once: waiting out that window is not a retry, it is a hang.
func TestRateLimitWithAFarOffResetIsNotRetried(t *testing.T) {
	calls := 0
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(24*time.Hour).Unix(), 10))
		w.WriteHeader(429)
		w.Write([]byte(`{"status":"error","error":{"code":"rate_limit_exceeded","message":"Too many NS group changes for this zone."}}`))
	})

	_, err := client.MoveZone(context.Background(), "xK9mQ2", "eu")
	require.Error(t, err)
	assert.True(t, IsRateLimited(err))
	assert.Equal(t, 1, calls, "a window a day out must not be slept through")
}
