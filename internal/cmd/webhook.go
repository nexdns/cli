package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nexdns/cli/internal/api"
)

// eventsHelp lists the subscribable event types. The API is authoritative and
// rejects an unknown one, so this is documentation rather than a client-side
// allowlist - a stale copy here must never be able to refuse an event the
// platform has started emitting.
const eventsHelp = `Event types:
  zone.created            zone.updated            zone.deleted
  record.created          record.updated          record.deleted
  dnssec.enabled          dnssec.disabled
  zone.health.problem     zone.health.resolved`

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage outbound event webhooks",
	Long: `Manage the webhook endpoints this account delivers events to.

Requires an API key with the webhooks.read / webhooks.write scopes.

` + eventsHelp,
}

var webhookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhooks",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}
		formatter, err := newFormatter(cmd)
		if err != nil {
			return err
		}

		webhooks, err := client.ListWebhooks(context.Background())
		if err != nil {
			return err
		}

		formatter.FormatWebhooks(webhooks)
		return nil
	},
}

var webhookShowCmd = &cobra.Command{
	Use:   "show <webhook-id>",
	Short: "Show a webhook with its recent deliveries",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}
		formatter, err := newFormatter(cmd)
		if err != nil {
			return err
		}

		webhook, err := client.GetWebhook(context.Background(), args[0])
		if err != nil {
			return err
		}

		formatter.FormatWebhook(webhook)
		return nil
	},
}

var webhookCreateCmd = &cobra.Command{
	Use:   "create <url>",
	Short: "Create a webhook",
	Long: `Create a webhook endpoint subscribed to one or more events.

The signing secret is printed once, on creation, and cannot be retrieved
afterwards - store it where your receiver can read it to verify the HMAC
signature of each delivery.

Examples:
  nexdns webhook create https://example.com/hook --events zone.created,zone.deleted
  nexdns webhook create https://example.com/hook --events record.created --description "prod"

` + eventsHelp,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]

		events, err := eventsFlag(cmd)
		if err != nil {
			return err
		}

		description, _ := cmd.Flags().GetString("description")

		if isDryRun(cmd) {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would create webhook %s for %s\n", url, strings.Join(events, ", "))
			return nil
		}

		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		created, err := client.CreateWebhook(context.Background(), api.CreateWebhookRequest{
			URL:         url,
			Events:      events,
			Description: description,
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Webhook created (ID: %s)\n", created.ID)
		fmt.Fprintf(cmd.OutOrStdout(), "Signing secret: %s\n", created.Secret)
		fmt.Fprintln(cmd.OutOrStdout(), "Store the secret now – it is not shown again.")
		return nil
	},
}

var webhookUpdateCmd = &cobra.Command{
	Use:   "update <webhook-id>",
	Short: "Update a webhook",
	Long: `Update a webhook's URL, events, description or active state.

The endpoint replaces the whole webhook rather than patching it, so this command
reads the current one first and re-sends every field you did not change. Use
--active=false to stop delivery without deleting the endpoint.

Examples:
  nexdns webhook update xK9mPq2R --events zone.created,zone.deleted,record.created
  nexdns webhook update xK9mPq2R --url https://example.com/hook-v2
  nexdns webhook update xK9mPq2R --active=false

` + eventsHelp,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		if !cmd.Flags().Changed("url") && !cmd.Flags().Changed("events") &&
			!cmd.Flags().Changed("description") && !cmd.Flags().Changed("active") {
			return fmt.Errorf("nothing to update: pass --url, --events, --description or --active")
		}

		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()
		current, err := client.GetWebhook(ctx, id)
		if err != nil {
			return err
		}

		req := api.UpdateWebhookRequest{
			URL:         current.URL,
			Events:      current.Events,
			Description: current.Description,
			IsActive:    current.IsActive,
		}

		if cmd.Flags().Changed("url") {
			req.URL, _ = cmd.Flags().GetString("url")
		}
		if cmd.Flags().Changed("events") {
			events, err := eventsFlag(cmd)
			if err != nil {
				return err
			}
			req.Events = events
		}
		if cmd.Flags().Changed("description") {
			req.Description, _ = cmd.Flags().GetString("description")
		}
		if cmd.Flags().Changed("active") {
			req.IsActive, _ = cmd.Flags().GetBool("active")
		}

		if isDryRun(cmd) {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would update webhook %s: %s, events %s, active %t\n",
				id, req.URL, strings.Join(req.Events, ", "), req.IsActive)
			return nil
		}

		formatter, err := newFormatter(cmd)
		if err != nil {
			return err
		}

		webhook, err := client.UpdateWebhook(ctx, id, req)
		if err != nil {
			return err
		}

		formatter.FormatWebhook(webhook)
		return nil
	},
}

var webhookDeleteCmd = &cobra.Command{
	Use:   "delete <webhook-id>",
	Short: "Delete a webhook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		ctx := context.Background()

		if isDryRun(cmd) {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would delete webhook %s\n", id)
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			webhook, err := client.GetWebhook(ctx, id)
			if err != nil {
				return err
			}
			if !confirmPrompt(fmt.Sprintf("Delete webhook %q (%s)?", webhook.URL, strings.Join(webhook.Events, ", "))) {
				fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
				return nil
			}
		}

		if err := client.DeleteWebhook(ctx, id); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Webhook deleted")
		return nil
	},
}

var webhookTestCmd = &cobra.Command{
	Use:   "test <webhook-id>",
	Short: "Queue a test event for a webhook",
	Long: `Queue a test event for delivery to the webhook.

The event is queued, not delivered synchronously: a success here means the
platform accepted it. Read the outcome back with "nexdns webhook show".`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		if isDryRun(cmd) {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] Would queue a test event for webhook %s\n", id)
			return nil
		}

		client, err := newAPIClient(cmd)
		if err != nil {
			return err
		}

		message, err := client.TestWebhook(context.Background(), id)
		if err != nil {
			return err
		}

		if message == "" {
			message = "Test event queued for delivery."
		}
		fmt.Fprintln(cmd.OutOrStdout(), message)
		return nil
	},
}

// eventsFlag reads --events as a comma-separated list, rejecting an empty one so
// the API is not asked to create a subscription to nothing.
func eventsFlag(cmd *cobra.Command) ([]string, error) {
	raw, _ := cmd.Flags().GetString("events")

	var events []string
	for _, e := range strings.Split(raw, ",") {
		if e = strings.TrimSpace(e); e != "" {
			events = append(events, e)
		}
	}

	if len(events) == 0 {
		return nil, fmt.Errorf("--events is required: a comma-separated list of event types")
	}

	return events, nil
}

func init() {
	webhookCreateCmd.Flags().String("events", "", "Comma-separated event types to subscribe to (required)")
	webhookCreateCmd.Flags().String("description", "", "Human-readable description")

	webhookUpdateCmd.Flags().String("url", "", "New endpoint URL")
	webhookUpdateCmd.Flags().String("events", "", "New comma-separated event types")
	webhookUpdateCmd.Flags().String("description", "", "New description")
	webhookUpdateCmd.Flags().Bool("active", true, "Whether the webhook receives deliveries")

	webhookDeleteCmd.Flags().Bool("force", false, "Skip the confirmation prompt")

	webhookCmd.AddCommand(webhookListCmd)
	webhookCmd.AddCommand(webhookShowCmd)
	webhookCmd.AddCommand(webhookCreateCmd)
	webhookCmd.AddCommand(webhookUpdateCmd)
	webhookCmd.AddCommand(webhookDeleteCmd)
	webhookCmd.AddCommand(webhookTestCmd)
	rootCmd.AddCommand(webhookCmd)
}
