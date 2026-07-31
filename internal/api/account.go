package api

import "context"

// GetAccount returns the current user's account info.
func (c *Client) GetAccount(ctx context.Context) (*Account, error) {
	var account Account
	err := c.do(ctx, "GET", "/account", nil, &account)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// ListAPIKeys returns all API keys for the current user.
func (c *Client) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	var keys []APIKey
	err := c.do(ctx, "GET", "/account/api-keys", nil, &keys)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// CreateAPIKey creates a new API key.
func (c *Client) CreateAPIKey(ctx context.Context, req CreateAPIKeyRequest) (*APIKeyWithSecret, error) {
	var key APIKeyWithSecret
	err := c.do(ctx, "POST", "/account/api-keys", req, &key)
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// DeleteAPIKey revokes an API key.
func (c *Client) DeleteAPIKey(ctx context.Context, id string) error {
	return c.do(ctx, "DELETE", "/account/api-keys/"+id, nil, nil)
}
