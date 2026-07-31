package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWebhooks(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/webhooks", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":[{"id":"xK9mPq2R","url":"https://example.com/hook","description":"prod","events":["zone.created"],"is_active":true,"failure_count":2,"last_triggered_at":"2026-07-29 12:00:00","created_at":"2026-07-01 10:00:00"}]}`))
	})

	webhooks, err := client.ListWebhooks(context.Background())
	require.NoError(t, err)
	require.Len(t, webhooks, 1)
	assert.Equal(t, "xK9mPq2R", webhooks[0].ID)
	assert.Equal(t, []string{"zone.created"}, webhooks[0].Events)
	assert.True(t, webhooks[0].IsActive)
	assert.Equal(t, 2, webhooks[0].FailureCount)
}

func TestGetWebhookCarriesDeliveries(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/webhooks/xK9mPq2R", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"id":"xK9mPq2R","url":"https://example.com/hook","events":["zone.created"],"is_active":true,"failure_count":0,"created_at":"2026-07-01 10:00:00","recent_deliveries":[{"event_id":"evt_1","event_type":"test","status":"failed","attempts":3,"last_status_code":405,"last_error":"HTTP 405","created_at":"2026-07-29 12:00:00"}]}}`))
	})

	webhook, err := client.GetWebhook(context.Background(), "xK9mPq2R")
	require.NoError(t, err)
	require.Len(t, webhook.RecentDeliveries, 1)
	assert.Equal(t, "failed", webhook.RecentDeliveries[0].Status)
	require.NotNil(t, webhook.RecentDeliveries[0].LastStatusCode)
	assert.Equal(t, 405, *webhook.RecentDeliveries[0].LastStatusCode)
	assert.Equal(t, "HTTP 405", webhook.RecentDeliveries[0].LastError)
}

func TestCreateWebhookReturnsTheSecretOnce(t *testing.T) {
	var body map[string]any
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		raw, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(raw, &body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write([]byte(`{"status":"success","data":{"id":"xK9mPq2R","secret":"whk_secret"}}`))
	})

	created, err := client.CreateWebhook(context.Background(), CreateWebhookRequest{
		URL:         "https://example.com/hook",
		Events:      []string{"zone.created", "record.created"},
		Description: "prod",
	})
	require.NoError(t, err)
	assert.Equal(t, "xK9mPq2R", created.ID)
	assert.Equal(t, "whk_secret", created.Secret)
	assert.Equal(t, "https://example.com/hook", body["url"])
	assert.Equal(t, []any{"zone.created", "record.created"}, body["events"])
	assert.Equal(t, "prod", body["description"])
}

// The update endpoint replaces rather than patches, so every field has to be on
// the wire even when the caller only meant to change one.
func TestUpdateWebhookSendsEveryField(t *testing.T) {
	var body map[string]any
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		raw, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(raw, &body))
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"id":"xK9mPq2R","url":"https://example.com/hook","events":["zone.created"],"is_active":false,"failure_count":0,"created_at":"2026-07-01 10:00:00"}}`))
	})

	webhook, err := client.UpdateWebhook(context.Background(), "xK9mPq2R", UpdateWebhookRequest{
		URL:      "https://example.com/hook",
		Events:   []string{"zone.created"},
		IsActive: false,
	})
	require.NoError(t, err)
	assert.False(t, webhook.IsActive)
	assert.Equal(t, false, body["is_active"])
	assert.Contains(t, body, "url")
	assert.Contains(t, body, "events")
}

func TestDeleteWebhook(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/webhooks/xK9mPq2R", r.URL.Path)
		w.WriteHeader(204)
	})

	assert.NoError(t, client.DeleteWebhook(context.Background(), "xK9mPq2R"))
}

func TestTestWebhookReturnsTheQueuedMessage(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/webhooks/xK9mPq2R/test", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"message":"Test event queued for delivery."}}`))
	})

	message, err := client.TestWebhook(context.Background(), "xK9mPq2R")
	require.NoError(t, err)
	assert.Equal(t, "Test event queued for delivery.", message)
}

func TestWebhookNotFoundIsAnAPIError(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		w.Write([]byte(`{"status":"error","error":{"code":"not_found","message":"Webhook not found."}}`))
	})

	_, err := client.GetWebhook(context.Background(), "ffffffff")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}
