package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRecords(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Contains(t, r.URL.Path, "/zones/abc/records")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":[{"id":"r1","name":"@","type":"A","content":"1.2.3.4","ttl":3600},{"id":"r2","name":"www","type":"CNAME","content":"example.com","ttl":3600}]}`))
	})

	records, err := client.ListRecords(context.Background(), "abc", ListRecordsOptions{})
	require.NoError(t, err)
	assert.Len(t, records, 2)
	assert.Equal(t, "A", records[0].Type)
	assert.Equal(t, "CNAME", records[1].Type)
}

func TestCreateRecordWithPriority(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		var req CreateRecordRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "MX", req.Type)
		assert.NotNil(t, req.Priority)
		assert.Equal(t, 10, *req.Priority)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write([]byte(`{"status":"success","data":{"id":"r3","name":"@","type":"MX","content":"mail.example.com","ttl":3600,"priority":10}}`))
	})

	priority := 10
	ttl := 3600
	record, err := client.CreateRecord(context.Background(), "abc", CreateRecordRequest{
		Name:     "@",
		Type:     "MX",
		Content:  "mail.example.com",
		TTL:      &ttl,
		Priority: &priority,
	})
	require.NoError(t, err)
	assert.Equal(t, "MX", record.Type)
	assert.Equal(t, 10, *record.Priority)
}

func TestUpdateRecord(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/zones/abc/records/r1", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"id":"r1","name":"@","type":"A","content":"5.6.7.8","ttl":600}}`))
	})

	content := "5.6.7.8"
	ttl := 600
	record, err := client.UpdateRecord(context.Background(), "abc", "r1", UpdateRecordRequest{
		Content: &content,
		TTL:     &ttl,
	})
	require.NoError(t, err)
	assert.Equal(t, "5.6.7.8", record.Content)
	assert.Equal(t, 600, record.TTL)
}

func TestDeleteRecord(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/zones/abc/records/r1", r.URL.Path)
		w.WriteHeader(204)
	})

	err := client.DeleteRecord(context.Background(), "abc", "r1")
	assert.NoError(t, err)
}
