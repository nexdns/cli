package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TTL is left unset unless --ttl was given.
func TestBuildCreateRecordRequestOmitsUnsetTTL(t *testing.T) {
	cmd := &cobra.Command{}
	addRecordFlags(cmd)

	req := buildCreateRecordRequest(cmd, "A", "www", "1.2.3.4")
	if req.TTL != nil {
		t.Errorf("expected no TTL in the request, got %d", *req.TTL)
	}

	if err := cmd.Flags().Set("ttl", "120"); err != nil {
		t.Fatalf("setting the flag: %v", err)
	}

	req = buildCreateRecordRequest(cmd, "A", "www", "1.2.3.4")
	if req.TTL == nil || *req.TTL != 120 {
		t.Errorf("expected TTL 120, got %v", req.TTL)
	}
}

// An out-of-range TTL is sent as given, for the API to reject.
func TestBuildCreateRecordRequestForwardsAnInvalidTTL(t *testing.T) {
	cmd := &cobra.Command{}
	addRecordFlags(cmd)

	if err := cmd.Flags().Set("ttl", "-5"); err != nil {
		t.Fatalf("setting the flag: %v", err)
	}

	req := buildCreateRecordRequest(cmd, "A", "www", "1.2.3.4")
	if req.TTL == nil || *req.TTL != -5 {
		t.Errorf("expected the TTL to be forwarded as -5, got %v", req.TTL)
	}
}
