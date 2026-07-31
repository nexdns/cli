package api

// PaginationMeta represents pagination metadata from list responses.
type PaginationMeta struct {
	Total    int `json:"total" yaml:"total"`
	Page     int `json:"page" yaml:"page"`
	PerPage  int `json:"per_page" yaml:"per_page"`
	LastPage int `json:"last_page" yaml:"last_page"`
}

// NSGroupInfo represents a nameserver group (detail view).
type NSGroupInfo struct {
	ID   string `json:"id" yaml:"id"`
	Slug string `json:"slug" yaml:"slug"`
	Name string `json:"name" yaml:"name"`
}

// SOA represents SOA record data.
type SOA struct {
	PrimaryNS  string `json:"primary_ns" yaml:"primary_ns"`
	AdminEmail string `json:"admin_email" yaml:"admin_email"`
	Serial     int64  `json:"serial" yaml:"serial"`
	Refresh    int    `json:"refresh" yaml:"refresh"`
	Retry      int    `json:"retry" yaml:"retry"`
	Expire     int    `json:"expire" yaml:"expire"`
	Minimum    int    `json:"minimum" yaml:"minimum"`
}

// ZoneListItem represents a zone in list responses (ns_group is a string).
type ZoneListItem struct {
	ID   string `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
	// UnicodeName is the native-script form of an internationalized name
	// (`тест.рф` for `xn--e1aybc.xn--p1ai`); equal to Name for ASCII zones.
	UnicodeName  string `json:"unicode_name,omitempty" yaml:"unicode_name,omitempty"`
	Type         string `json:"type" yaml:"type"`
	Status       string `json:"status" yaml:"status"`
	NSGroup      string `json:"ns_group" yaml:"ns_group"`
	RecordsCount int    `json:"records_count" yaml:"records_count"`
	CreatedAt    string `json:"created_at" yaml:"created_at"`
	UpdatedAt    string `json:"updated_at" yaml:"updated_at"`
}

// Zone represents a zone in detail responses (ns_group is an object).
type Zone struct {
	ID   string `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
	// UnicodeName is the native-script form of an internationalized name.
	UnicodeName  string       `json:"unicode_name,omitempty" yaml:"unicode_name,omitempty"`
	Type         string       `json:"type" yaml:"type"`
	Status       string       `json:"status" yaml:"status"`
	NSGroup      *NSGroupInfo `json:"ns_group" yaml:"ns_group"`
	RecordsCount int          `json:"records_count" yaml:"records_count"`
	SOA          *SOA         `json:"soa" yaml:"soa"`
	Nameservers  []string     `json:"nameservers" yaml:"nameservers"`
	CreatedAt    string       `json:"created_at" yaml:"created_at"`
	UpdatedAt    string       `json:"updated_at" yaml:"updated_at"`
}

// Record represents a DNS record.
type Record struct {
	ID   string `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
	// UnicodeName is the native-script form of an internationalized label.
	UnicodeName string                 `json:"unicode_name,omitempty" yaml:"unicode_name,omitempty"`
	Type        string                 `json:"type" yaml:"type"`
	Content     string                 `json:"content" yaml:"content"`
	TTL         int                    `json:"ttl" yaml:"ttl"`
	Priority    *int                   `json:"priority,omitempty" yaml:"priority,omitempty"`
	Weight      *int                   `json:"weight,omitempty" yaml:"weight,omitempty"`
	Port        *int                   `json:"port,omitempty" yaml:"port,omitempty"`
	Flags       *int                   `json:"flags,omitempty" yaml:"flags,omitempty"`
	Tag         string                 `json:"tag,omitempty" yaml:"tag,omitempty"`
	Disabled    bool                   `json:"disabled" yaml:"disabled"`
	Fields      map[string]interface{} `json:"fields,omitempty" yaml:"fields,omitempty"`
}

// Webhook represents an outbound event subscription: one of the caller's own
// endpoints plus the events it is subscribed to. The signing secret is not part
// of it - the API returns that once, on creation ({@see CreatedWebhook}).
type Webhook struct {
	ID          string   `json:"id" yaml:"id"`
	URL         string   `json:"url" yaml:"url"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Events      []string `json:"events" yaml:"events"`
	IsActive    bool     `json:"is_active" yaml:"is_active"`
	// FailureCount is the run of consecutive delivery failures; the platform
	// deactivates an endpoint that keeps failing.
	FailureCount    int    `json:"failure_count" yaml:"failure_count"`
	LastTriggeredAt string `json:"last_triggered_at,omitempty" yaml:"last_triggered_at,omitempty"`
	CreatedAt       string `json:"created_at" yaml:"created_at"`
	UpdatedAt       string `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
	// RecentDeliveries is populated by the detail endpoint only (the 10 most
	// recent attempts), and is empty in a list response.
	RecentDeliveries []WebhookDelivery `json:"recent_deliveries,omitempty" yaml:"recent_deliveries,omitempty"`
}

// WebhookDelivery represents one delivery attempt of one event.
type WebhookDelivery struct {
	EventID        string `json:"event_id" yaml:"event_id"`
	EventType      string `json:"event_type" yaml:"event_type"`
	Status         string `json:"status" yaml:"status"`
	Attempts       int    `json:"attempts" yaml:"attempts"`
	LastStatusCode *int   `json:"last_status_code,omitempty" yaml:"last_status_code,omitempty"`
	LastError      string `json:"last_error,omitempty" yaml:"last_error,omitempty"`
	CreatedAt      string `json:"created_at" yaml:"created_at"`
	DeliveredAt    string `json:"delivered_at,omitempty" yaml:"delivered_at,omitempty"`
}

// CreatedWebhook is the create response: the new id and the signing secret, which
// is returned exactly once and cannot be retrieved afterwards.
type CreatedWebhook struct {
	ID     string `json:"id" yaml:"id"`
	Secret string `json:"secret" yaml:"secret"`
}

// CreateWebhookRequest represents a request to create a webhook.
type CreateWebhookRequest struct {
	URL         string   `json:"url" yaml:"url"`
	Events      []string `json:"events" yaml:"events"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

// UpdateWebhookRequest represents a request to update a webhook. The endpoint is
// a full replace, not a patch: `url` and `events` are always required, so a
// caller changing one field sends the others as they stand.
type UpdateWebhookRequest struct {
	URL         string   `json:"url" yaml:"url"`
	Events      []string `json:"events" yaml:"events"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	IsActive    bool     `json:"is_active" yaml:"is_active"`
}

// DNSSECKey represents a DNSSEC key.
type DNSSECKey struct {
	Type      string   `json:"type" yaml:"type"`
	Algorithm string   `json:"algorithm" yaml:"algorithm"`
	Bits      int      `json:"bits" yaml:"bits"`
	Active    bool     `json:"active" yaml:"active"`
	Published bool     `json:"published" yaml:"published"`
	DNSKEY    string   `json:"dnskey" yaml:"dnskey"`
	DS        []string `json:"ds" yaml:"ds"`
}

// DNSSECStatus represents DNSSEC status for a zone.
type DNSSECStatus struct {
	Enabled   bool        `json:"enabled" yaml:"enabled"`
	Keys      []DNSSECKey `json:"keys" yaml:"keys"`
	DSRecords []string    `json:"ds_records" yaml:"ds_records"`
}

// Account represents user account info.
type Account struct {
	ID           string               `json:"id" yaml:"id"`
	Email        string               `json:"email" yaml:"email"`
	Name         string               `json:"name" yaml:"name"`
	Status       string               `json:"status" yaml:"status"`
	Language     string               `json:"language" yaml:"language"`
	Timezone     string               `json:"timezone" yaml:"timezone"`
	CreatedAt    string               `json:"created_at" yaml:"created_at"`
	Subscription *AccountSubscription `json:"subscription,omitempty" yaml:"subscription,omitempty"`
}

// AccountSubscription represents subscription info within account.
type AccountSubscription struct {
	Plan               string `json:"plan" yaml:"plan"`
	BillingCycle       string `json:"billing_cycle" yaml:"billing_cycle"`
	Status             string `json:"status" yaml:"status"`
	CurrentPeriodStart string `json:"current_period_start" yaml:"current_period_start"`
	CurrentPeriodEnd   string `json:"current_period_end" yaml:"current_period_end"`
}

// APIKey represents an API key (without the full secret).
type APIKey struct {
	ID          string   `json:"id" yaml:"id"`
	Name        string   `json:"name" yaml:"name"`
	KeyPrefix   string   `json:"key_prefix" yaml:"key_prefix"`
	Permissions []string `json:"permissions" yaml:"permissions"`
	LastUsedAt  *string  `json:"last_used_at,omitempty" yaml:"last_used_at,omitempty"`
	ExpiresAt   *string  `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	CreatedAt   string   `json:"created_at" yaml:"created_at"`
}

// APIKeyWithSecret represents an API key with the full secret (returned on creation only).
type APIKeyWithSecret struct {
	APIKey
	Key string `json:"key" yaml:"key"`
}

// Subscription represents billing subscription.
type Subscription struct {
	ID                 string `json:"id" yaml:"id"`
	Plan               string `json:"plan" yaml:"plan"`
	BillingCycle       string `json:"billing_cycle" yaml:"billing_cycle"`
	Status             string `json:"status" yaml:"status"`
	CurrentPeriodStart string `json:"current_period_start" yaml:"current_period_start"`
	CurrentPeriodEnd   string `json:"current_period_end" yaml:"current_period_end"`
	CreatedAt          string `json:"created_at" yaml:"created_at"`
}

// Plan represents a billing plan.
type Plan struct {
	ID           string   `json:"id" yaml:"id"`
	Name         string   `json:"name" yaml:"name"`
	Slug         string   `json:"slug" yaml:"slug"`
	Description  string   `json:"description" yaml:"description"`
	PriceMonthly float64  `json:"price_monthly" yaml:"price_monthly"`
	PriceYearly  float64  `json:"price_yearly" yaml:"price_yearly"`
	Currency     string   `json:"currency" yaml:"currency"`
	MaxDomains   int      `json:"max_domains" yaml:"max_domains"`
	MaxRecords   int      `json:"max_records" yaml:"max_records"`
	Features     []string `json:"features" yaml:"features"`
}

// Invoice represents a billing invoice.
type Invoice struct {
	ID        string  `json:"id" yaml:"id"`
	Number    string  `json:"number" yaml:"number"`
	Status    string  `json:"status" yaml:"status"`
	Amount    float64 `json:"amount" yaml:"amount"`
	Currency  string  `json:"currency" yaml:"currency"`
	IssuedAt  string  `json:"issued_at" yaml:"issued_at"`
	DueDate   string  `json:"due_date" yaml:"due_date"`
	PaidAt    *string `json:"paid_at,omitempty" yaml:"paid_at,omitempty"`
	CreatedAt string  `json:"created_at" yaml:"created_at"`
}

// NSGroup represents a public nameserver group.
type NSGroup struct {
	ID   string `json:"id" yaml:"id"`
	Name string `json:"name" yaml:"name"`
	Slug string `json:"slug" yaml:"slug"`
}

// Request types

// CreateZoneRequest represents a request to create a zone.
type CreateZoneRequest struct {
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type,omitempty" yaml:"type,omitempty"`
	NSGroup  string `json:"ns_group,omitempty" yaml:"ns_group,omitempty"`
	MasterIP string `json:"master_ip,omitempty" yaml:"master_ip,omitempty"`
}

// MoveZoneRequest represents a request to move a zone into another nameserver
// group. The group is named by its slug, the only identifier the API accepts.
type MoveZoneRequest struct {
	NSGroup string `json:"ns_group" yaml:"ns_group"`
}

// CreateRecordRequest represents a request to create a DNS record.
type CreateRecordRequest struct {
	Name    string `json:"name" yaml:"name"`
	Type    string `json:"type" yaml:"type"`
	Content string `json:"content" yaml:"content"`
	// TTL is a pointer so an absent TTL and a deliberate zero are distinct: the
	// API keeps the rrset's own TTL when the field is missing, and a plain int
	// with omitempty could express neither 0 nor a value the API should reject.
	TTL          *int   `json:"ttl,omitempty" yaml:"ttl,omitempty"`
	Priority     *int   `json:"priority,omitempty" yaml:"priority,omitempty"`
	Weight       *int   `json:"weight,omitempty" yaml:"weight,omitempty"`
	Port         *int   `json:"port,omitempty" yaml:"port,omitempty"`
	Flags        *int   `json:"flags,omitempty" yaml:"flags,omitempty"`
	Tag          string `json:"tag,omitempty" yaml:"tag,omitempty"`
	KeyTag       *int   `json:"keytag,omitempty" yaml:"keytag,omitempty"`
	Algorithm    *int   `json:"algorithm,omitempty" yaml:"algorithm,omitempty"`
	DigestType   *int   `json:"digest_type,omitempty" yaml:"digest_type,omitempty"`
	Usage        *int   `json:"usage,omitempty" yaml:"usage,omitempty"`
	Selector     *int   `json:"selector,omitempty" yaml:"selector,omitempty"`
	MatchingType *int   `json:"matching_type,omitempty" yaml:"matching_type,omitempty"`
}

// UpdateRecordRequest represents a request to update a DNS record.
type UpdateRecordRequest struct {
	Name     *string `json:"name,omitempty" yaml:"name,omitempty"`
	Content  *string `json:"content,omitempty" yaml:"content,omitempty"`
	TTL      *int    `json:"ttl,omitempty" yaml:"ttl,omitempty"`
	Priority *int    `json:"priority,omitempty" yaml:"priority,omitempty"`
}

// CreateAPIKeyRequest represents a request to create an API key.
type CreateAPIKeyRequest struct {
	Name        string   `json:"name" yaml:"name"`
	Permissions []string `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	ExpiresAt   string   `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
}

// Option types for list operations

// ListZonesOptions represents options for listing zones.
type ListZonesOptions struct {
	Page    int
	PerPage int
	Search  string
}

// ListRecordsOptions represents options for listing records.
type ListRecordsOptions struct {
	Type   string
	Name   string
	Search string
}

// ListInvoicesOptions represents options for listing invoices.
type ListInvoicesOptions struct {
	Page    int
	PerPage int
	Status  string
}

// Response wrappers

// ZoneListResponse represents a paginated zone list response.
type ZoneListResponse struct {
	Data []ZoneListItem  `json:"data" yaml:"data"`
	Meta *PaginationMeta `json:"meta" yaml:"meta"`
}

// InvoiceListResponse represents a paginated invoice list response.
type InvoiceListResponse struct {
	Data []Invoice       `json:"data" yaml:"data"`
	Meta *PaginationMeta `json:"meta" yaml:"meta"`
}
