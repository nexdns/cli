package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// ListZones returns a paginated list of zones.
func (c *Client) ListZones(ctx context.Context, opts ListZonesOptions) (*ZoneListResponse, error) {
	params := map[string]string{}
	if opts.Page > 0 {
		params["page"] = strconv.Itoa(opts.Page)
	}
	if opts.PerPage > 0 {
		params["per_page"] = strconv.Itoa(opts.PerPage)
	}
	if opts.Search != "" {
		params["search"] = opts.Search
	}

	var data []ZoneListItem
	var meta PaginationMeta
	err := c.doWithMeta(ctx, "GET", "/zones"+buildQuery(params), nil, &data, &meta)
	if err != nil {
		return nil, err
	}
	return &ZoneListResponse{Data: data, Meta: &meta}, nil
}

// ListAllZones fetches all zones across all pages.
func (c *Client) ListAllZones(ctx context.Context, opts ListZonesOptions) ([]ZoneListItem, error) {
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}

	var all []ZoneListItem
	for {
		resp, err := c.ListZones(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Data...)
		if resp.Meta == nil || opts.Page >= resp.Meta.LastPage {
			break
		}
		opts.Page++
	}
	return all, nil
}

// GetZone returns a zone by its public ID.
func (c *Client) GetZone(ctx context.Context, id string) (*Zone, error) {
	var zone Zone
	err := c.do(ctx, "GET", "/zones/"+id, nil, &zone)
	if err != nil {
		return nil, err
	}
	return &zone, nil
}

// GetZoneByName searches for a zone by exact domain name.
func (c *Client) GetZoneByName(ctx context.Context, name string) (*ZoneListItem, error) {
	resp, err := c.ListZones(ctx, ListZonesOptions{Search: name, PerPage: 100})
	if err != nil {
		return nil, err
	}

	// Match either the ACE or the Unicode form of an IDN zone.
	for i := range resp.Data {
		if resp.Data[i].Name == name || resp.Data[i].UnicodeName == name {
			return &resp.Data[i], nil
		}
	}

	return nil, &APIError{
		StatusCode: 404,
		Code:       "not_found",
		Message:    fmt.Sprintf("Zone %q not found", name),
	}
}

// CreateZone creates a new zone.
func (c *Client) CreateZone(ctx context.Context, req CreateZoneRequest) (*ZoneListItem, error) {
	var zone ZoneListItem
	err := c.do(ctx, "POST", "/zones", req, &zone)
	if err != nil {
		return nil, err
	}
	return &zone, nil
}

// DeleteZone deletes a zone by its public ID.
func (c *Client) DeleteZone(ctx context.Context, id string) error {
	return c.do(ctx, "DELETE", "/zones/"+id, nil, nil)
}

// MoveZone moves a zone into another nameserver group, identified by slug,
// and returns the updated zone.
func (c *Client) MoveZone(ctx context.Context, id, nsGroupSlug string) (*Zone, error) {
	var zone Zone
	err := c.do(ctx, "PATCH", "/zones/"+id, MoveZoneRequest{NSGroup: nsGroupSlug}, &zone)
	if err != nil {
		return nil, err
	}
	return &zone, nil
}

// ListNSGroups returns all active NS groups.
func (c *Client) ListNSGroups(ctx context.Context) ([]NSGroup, error) {
	var groups []NSGroup
	err := c.do(ctx, "GET", "/ns-groups", nil, &groups)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

// ExportZone exports a zone as "bind" (zone file text) or "json".
func (c *Client) ExportZone(ctx context.Context, id, format string) (string, error) {
	params := map[string]string{}
	if format != "" {
		params["format"] = format
	}

	var raw json.RawMessage
	err := c.do(ctx, "GET", "/zones/"+id+"/export"+buildQuery(params), nil, &raw)
	if err != nil {
		return "", err
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return "", fmt.Errorf("parsing export payload: %w", err)
		}
		return text, nil
	}

	var buf bytes.Buffer
	if err := json.Indent(&buf, trimmed, "", "  "); err != nil {
		return string(trimmed), nil
	}
	buf.WriteByte('\n')

	return buf.String(), nil
}
