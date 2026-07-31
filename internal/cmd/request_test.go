package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// buildCreateRecordRequest must leave TTL unset unless --ttl was given: a TTL is
// a property of the whole rrset, so sending the flag's default alongside a new
// value retimed every value already at that name.
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

// An out-of-range TTL is sent as given so the API can refuse it. Dropping it
// client-side reported success while creating the record with the default TTL,
// telling the user the opposite of what happened.
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
