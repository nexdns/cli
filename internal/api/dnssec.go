package api

import "context"

// GetDNSSEC returns the DNSSEC status for a zone.
func (c *Client) GetDNSSEC(ctx context.Context, zoneID string) (*DNSSECStatus, error) {
	var status DNSSECStatus
	err := c.do(ctx, "GET", "/zones/"+zoneID+"/dnssec", nil, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// EnableDNSSEC enables DNSSEC for a zone.
func (c *Client) EnableDNSSEC(ctx context.Context, zoneID string) (*DNSSECStatus, error) {
	var status DNSSECStatus
	err := c.do(ctx, "POST", "/zones/"+zoneID+"/dnssec/enable", nil, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// DisableDNSSEC disables DNSSEC for a zone.
func (c *Client) DisableDNSSEC(ctx context.Context, zoneID string) error {
	return c.do(ctx, "POST", "/zones/"+zoneID+"/dnssec/disable", nil, nil)
}
