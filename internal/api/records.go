package api

import (
	"context"
)

// ListRecords returns all records for a zone (no pagination).
func (c *Client) ListRecords(ctx context.Context, zoneID string, opts ListRecordsOptions) ([]Record, error) {
	params := map[string]string{}
	if opts.Type != "" {
		params["type"] = opts.Type
	}
	if opts.Name != "" {
		params["name"] = opts.Name
	}
	if opts.Search != "" {
		params["search"] = opts.Search
	}

	var records []Record
	err := c.do(ctx, "GET", "/zones/"+zoneID+"/records"+buildQuery(params), nil, &records)
	if err != nil {
		return nil, err
	}
	return records, nil
}

// GetRecord returns a single record.
func (c *Client) GetRecord(ctx context.Context, zoneID, recordID string) (*Record, error) {
	var record Record
	err := c.do(ctx, "GET", "/zones/"+zoneID+"/records/"+recordID, nil, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// CreateRecord creates a new DNS record.
func (c *Client) CreateRecord(ctx context.Context, zoneID string, req CreateRecordRequest) (*Record, error) {
	var record Record
	err := c.do(ctx, "POST", "/zones/"+zoneID+"/records", req, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateRecord updates an existing DNS record.
func (c *Client) UpdateRecord(ctx context.Context, zoneID, recordID string, req UpdateRecordRequest) (*Record, error) {
	var record Record
	err := c.do(ctx, "PUT", "/zones/"+zoneID+"/records/"+recordID, req, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// DeleteRecord deletes a DNS record.
func (c *Client) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	return c.do(ctx, "DELETE", "/zones/"+zoneID+"/records/"+recordID, nil, nil)
}
