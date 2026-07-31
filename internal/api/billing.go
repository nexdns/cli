package api

import (
	"context"
	"strconv"
)

// GetSubscription returns the current subscription.
func (c *Client) GetSubscription(ctx context.Context) (*Subscription, error) {
	var sub Subscription
	err := c.do(ctx, "GET", "/billing/subscription", nil, &sub)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// ListPlans returns all available billing plans.
func (c *Client) ListPlans(ctx context.Context) ([]Plan, error) {
	var plans []Plan
	err := c.do(ctx, "GET", "/billing/plans", nil, &plans)
	if err != nil {
		return nil, err
	}
	return plans, nil
}

// ListInvoices returns a paginated list of invoices.
func (c *Client) ListInvoices(ctx context.Context, opts ListInvoicesOptions) (*InvoiceListResponse, error) {
	params := map[string]string{}
	if opts.Page > 0 {
		params["page"] = strconv.Itoa(opts.Page)
	}
	if opts.PerPage > 0 {
		params["per_page"] = strconv.Itoa(opts.PerPage)
	}
	if opts.Status != "" {
		params["status"] = opts.Status
	}

	var data []Invoice
	var meta PaginationMeta
	err := c.doWithMeta(ctx, "GET", "/billing/invoices"+buildQuery(params), nil, &data, &meta)
	if err != nil {
		return nil, err
	}
	return &InvoiceListResponse{Data: data, Meta: &meta}, nil
}

// GetInvoice returns an invoice by ID or number.
func (c *Client) GetInvoice(ctx context.Context, id string) (*Invoice, error) {
	var invoice Invoice
	err := c.do(ctx, "GET", "/billing/invoices/"+id, nil, &invoice)
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}
