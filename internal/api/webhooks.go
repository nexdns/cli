package api

import "context"

// ListWebhooks returns the caller's webhooks. Requires the `webhooks.read` scope.
func (c *Client) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	var webhooks []Webhook
	err := c.do(ctx, "GET", "/webhooks", nil, &webhooks)
	if err != nil {
		return nil, err
	}
	return webhooks, nil
}

// GetWebhook returns one webhook by its public ID, including its 10 most recent
// delivery attempts. Requires the `webhooks.read` scope.
func (c *Client) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	var webhook Webhook
	err := c.do(ctx, "GET", "/webhooks/"+id, nil, &webhook)
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

// CreateWebhook registers a webhook and returns its id together with the signing
// secret. The secret exists in this response only: it is never returned again, so
// a caller that does not keep it has to replace the webhook to get a new one.
// Requires the `webhooks.write` scope.
func (c *Client) CreateWebhook(ctx context.Context, req CreateWebhookRequest) (*CreatedWebhook, error) {
	var created CreatedWebhook
	err := c.do(ctx, "POST", "/webhooks", req, &created)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdateWebhook replaces a webhook's URL, events, description and active flag,
// and returns the stored result. Requires the `webhooks.write` scope.
func (c *Client) UpdateWebhook(ctx context.Context, id string, req UpdateWebhookRequest) (*Webhook, error) {
	var webhook Webhook
	err := c.do(ctx, "PUT", "/webhooks/"+id, req, &webhook)
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

// DeleteWebhook removes a webhook. Requires the `webhooks.write` scope.
func (c *Client) DeleteWebhook(ctx context.Context, id string) error {
	return c.do(ctx, "DELETE", "/webhooks/"+id, nil, nil)
}

// TestWebhook queues a test event for the webhook. The call returns as soon as the
// event is queued, so a 200 means "accepted for delivery", not "delivered" - read
// the outcome back with GetWebhook. Requires the `webhooks.write` scope.
func (c *Client) TestWebhook(ctx context.Context, id string) (string, error) {
	var result struct {
		Message string `json:"message"`
	}
	if err := c.do(ctx, "POST", "/webhooks/"+id+"/test", nil, &result); err != nil {
		return "", err
	}
	return result.Message, nil
}
