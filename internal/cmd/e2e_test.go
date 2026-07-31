package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build binary for E2E tests
	dir, _ := os.MkdirTemp("", "nexdns-e2e-*")
	binaryPath = filepath.Join(dir, "nexdns")

	build := exec.Command("go", "build", "-o", binaryPath, "../../cmd/nexdns")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func runCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run CLI: %v", err)
		}
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// --- Version ---

func TestE2E_Version(t *testing.T) {
	stdout, _, code := runCLI(t, "version")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "nexdns version") {
		t.Errorf("expected 'nexdns version' in output, got: %s", stdout)
	}
}

func TestE2E_VersionFlag(t *testing.T) {
	stdout, _, code := runCLI(t, "-V")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Error("expected version output, got empty")
	}
}

// --- Exit codes ---

func TestE2E_UnknownCommand_Exit2(t *testing.T) {
	_, _, code := runCLI(t, "nonexistent")
	if code != 2 {
		t.Errorf("expected exit 2 for unknown command, got %d", code)
	}
}

func TestE2E_MissingArgs_Exit2(t *testing.T) {
	_, _, code := runCLI(t, "zone", "add")
	if code != 2 {
		t.Errorf("expected exit 2 for missing args, got %d", code)
	}
}

func TestE2E_UnknownFlag_Exit2(t *testing.T) {
	_, _, code := runCLI(t, "zone", "list", "--nonexistent")
	if code != 2 {
		t.Errorf("expected exit 2 for unknown flag, got %d", code)
	}
}

// A typo'd subcommand of a group used to print the group's help and exit 0,
// so a script could not tell it from a completed operation.
func TestE2E_UnknownSubcommand_Exit2(t *testing.T) {
	for _, group := range []string{"zone", "record", "dnssec", "account", "config", "webhook"} {
		_, _, code := runCLI(t, group, "frobnicate")
		if code != 2 {
			t.Errorf("expected exit 2 for unknown %s subcommand, got %d", group, code)
		}
	}
}

// A group invoked with no subcommand at all is still just a request for help.
func TestE2E_GroupWithoutSubcommand_Exit0(t *testing.T) {
	stdout, _, code := runCLI(t, "zone")
	if code != 0 {
		t.Errorf("expected exit 0 for a bare group command, got %d", code)
	}
	if !strings.Contains(stdout, "Available Commands") {
		t.Errorf("expected the group help on stdout, got %q", stdout)
	}
}

// --- Auth errors ---

func TestE2E_NoToken_Exit1(t *testing.T) {
	_, stderr, code := runCLI(t, "zone", "list", "--config", "/tmp/nexdns-nonexistent-config.yaml")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "not authenticated") {
		t.Errorf("expected auth error message, got: %s", stderr)
	}
}

func TestE2E_InvalidTokenFormat(t *testing.T) {
	_, stderr, code := runCLI(t, "auth", "token", "invalid_token")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "nxd_") {
		t.Errorf("expected format hint, got: %s", stderr)
	}
}

func TestE2E_TokenTooShort(t *testing.T) {
	_, stderr, code := runCLI(t, "auth", "token", "nxd_abc")
	if code != 1 {
		t.Errorf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "too short") {
		t.Errorf("expected 'too short' error, got: %s", stderr)
	}
}

// --- Help ---

func TestE2E_Help(t *testing.T) {
	stdout, _, code := runCLI(t, "--help")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	for _, cmd := range []string{"zone", "record", "dnssec", "account", "auth", "apply", "diff", "pull", "version", "completion"} {
		if !strings.Contains(stdout, cmd) {
			t.Errorf("expected %q in help output", cmd)
		}
	}
}

func TestE2E_ZoneHelp(t *testing.T) {
	stdout, _, code := runCLI(t, "zone", "--help")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	for _, sub := range []string{"list", "add", "delete", "info", "export", "import", "ensure", "check"} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("expected %q in zone help", sub)
		}
	}
}

func TestE2E_RecordHelp(t *testing.T) {
	stdout, _, code := runCLI(t, "record", "--help")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	for _, sub := range []string{"list", "add", "update", "delete", "ensure"} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("expected %q in record help", sub)
		}
	}
}

// --- Completion ---

func TestE2E_CompletionBash(t *testing.T) {
	stdout, _, code := runCLI(t, "completion", "bash")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "bash completion") {
		t.Error("expected bash completion script")
	}
}

func TestE2E_CompletionZsh(t *testing.T) {
	stdout, _, code := runCLI(t, "completion", "zsh")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "compdef") {
		t.Error("expected zsh completion script")
	}
}

// --- Dry-run ---

func TestE2E_RecordAddDryRun(t *testing.T) {
	stdout, _, code := runCLI(t, "record", "add", "example.com", "A", "www", "1.2.3.4", "--dry-run", "--config", "/tmp/nexdns-nonexistent.yaml")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "[dry-run]") {
		t.Errorf("expected dry-run message, got: %s", stdout)
	}
}

// --- Auth save/load cycle ---

func TestE2E_AuthLogoutNonexistentConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	stdout, _, code := runCLI(t, "auth", "logout", "--config", cfgPath)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "Token removed") {
		t.Errorf("expected logout message, got: %s", stdout)
	}
}

// --- Webhooks and the NS-group move ---

// --events is required, and the check happens before any network call, so the
// error is the same with no config at all.
func TestE2E_WebhookCreateRequiresEvents(t *testing.T) {
	_, stderr, code := runCLI(t, "webhook", "create", "https://example.com/hook", "--config", "/tmp/nexdns-nonexistent.yaml")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "--events is required") {
		t.Errorf("expected the missing-events error, got: %s", stderr)
	}
}

func TestE2E_WebhookCreateDryRun(t *testing.T) {
	stdout, _, code := runCLI(t, "webhook", "create", "https://example.com/hook", "--events", "zone.created", "--dry-run", "--config", "/tmp/nexdns-nonexistent.yaml")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "[dry-run]") {
		t.Errorf("expected dry-run message, got: %s", stdout)
	}
}

// An update that names no field is a usage mistake, caught before the fetch the
// replace-semantics workaround would otherwise perform.
func TestE2E_WebhookUpdateWithoutFieldsFails(t *testing.T) {
	_, stderr, code := runCLI(t, "webhook", "update", "xK9mPq2R", "--config", "/tmp/nexdns-nonexistent.yaml")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "nothing to update") {
		t.Errorf("expected the nothing-to-update error, got: %s", stderr)
	}
}

func TestE2E_ZoneMoveRequiresTwoArgs(t *testing.T) {
	_, _, code := runCLI(t, "zone", "move", "example.com", "--config", "/tmp/nexdns-nonexistent.yaml")
	if code != 2 {
		t.Errorf("expected exit 2 for a missing NS group, got %d", code)
	}
}

func TestE2E_WebhookAndMoveAppearInHelp(t *testing.T) {
	stdout, _, code := runCLI(t, "--help")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "webhook") {
		t.Errorf("expected the webhook group in the root help, got: %s", stdout)
	}

	stdout, _, code = runCLI(t, "zone", "--help")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "move") {
		t.Errorf("expected zone move in the zone help, got: %s", stdout)
	}
}
