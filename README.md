# NexDNS CLI

[![Latest release](https://img.shields.io/github/v/release/nexdns/cli?sort=semver&label=release)](https://github.com/nexdns/cli/releases)
[![License](https://img.shields.io/github/license/nexdns/cli)](LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/nexdns/cli)](go.mod)
[![Docker pulls](https://img.shields.io/docker/pulls/nexdns/cli)](https://hub.docker.com/r/nexdns/cli)

`nexdns` is the official command-line tool for [NexDNS](https://nexdns.tech) – a managed DNS hosting platform with a REST API, managed DNSSEC, outbound event webhooks, and ISPmanager / DNSmanager API compatibility.

It is a single static binary that talks to the NexDNS REST API. Everything you can do to a zone in the panel you can do from a shell, a Makefile, or a CI job: create zones, edit records, import a BIND zone file, turn on DNSSEC and read back the DS records for your registrar, reconcile a whole account against a YAML file, and answer ACME DNS-01 challenges while a certificate is issued.

**Who it is for**

- **DevOps and platform engineers** who want DNS in the same review-and-merge flow as the rest of their infrastructure.
- **CI/CD pipelines** – no interactive prompts, machine-readable output on every data command, and exit codes that distinguish a usage mistake from a failed operation.
- **Anyone migrating in** from another provider: export a BIND zone file there, `nexdns zone import` here, verify with `nexdns zone check`, then repoint the delegation.
- **Certificate automation** – wildcard certificates via DNS-01 without handing an ACME client a full-access key.

**What you need**

An account on a plan that includes API access, and an API key created at [nexdns.tech/settings/api-keys](https://nexdns.tech/settings/api-keys). DNSSEC, webhooks, secondary (slave) zones and white-label nameservers start above the entry plan; reverse zones are on every plan. The [pricing page](https://nexdns.tech/pricing) has the current matrix.

## Contents

- [Installation](#installation)
- [Quick start](#quick-start)
- [Command reference](#command-reference)
- [Zones](#zones) · [Records](#records) · [DNSSEC](#dnssec) · [Webhooks](#webhooks)
- [DNS-as-Code](#dns-as-code)
- [ACME / certificates](#acme--certificates)
- [Output formats and scripting](#output-formats-and-scripting)
- [Configuration](#configuration)
- [Migrating from another provider](#migrating-from-another-provider)
- [Troubleshooting](#troubleshooting)
- [Related projects](#related-projects)

## Installation

### Install script (Linux, macOS)

```bash
curl -sL https://get.nexdns.tech/cli | sh
```

Detects your platform, downloads the matching release archive, verifies it against the `checksums.txt` published with the release, and installs to `/usr/local/bin`. Read it first with `curl https://get.nexdns.tech/cli` if you would rather not pipe an unread script to a shell - the same file is in this repository at [`scripts/install.sh`](scripts/install.sh).

### Homebrew (macOS, Linux)

```bash
brew tap nexdns/tap
brew install --cask nexdns-cli
```

### Go install

Requires Go 1.26 or later.

```bash
go install github.com/nexdns/cli/cmd/nexdns@latest
```

### Prebuilt archive

Every release publishes `nexdns_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows) plus a `checksums.txt`, on the [releases page](https://github.com/nexdns/cli/releases).

```bash
curl -LO https://github.com/nexdns/cli/releases/download/v1.0.0/nexdns_1.0.0_linux_amd64.tar.gz
tar -xzf nexdns_1.0.0_linux_amd64.tar.gz
sudo mv nexdns /usr/local/bin/
nexdns version
```

Builds are published for Linux (`amd64`, `arm64`, `386`), macOS (`amd64`, `arm64`) and Windows (`amd64`, `arm64`).

### Docker

The image is `nexdns/cli`, tagged `latest` and per release, for `linux/amd64` and `linux/arm64`. The binary is the entrypoint, so append the command you want.

```bash
docker run --rm -e NEXDNS_TOKEN=nxd_exampletoken1234567890 nexdns/cli zone list
```

### From source

```bash
git clone https://github.com/nexdns/cli && cd cli
make build          # -> bin/nexdns
```

### Shell completion

```bash
nexdns completion bash > /etc/bash_completion.d/nexdns
nexdns completion zsh  > "${fpath[1]}/_nexdns"
nexdns completion fish > ~/.config/fish/completions/nexdns.fish
```

`nexdns completion --help` covers PowerShell and per-shell loading.

## Quick start

```bash
# 1. Authenticate. The token is checked against the API before it is stored.
nexdns auth token nxd_exampletoken1234567890
# Token verified. Authenticated as you@example.com
# Token saved to /home/you/.nexdns/config.yaml

# 2. Create a zone and look at what was provisioned.
nexdns zone add example.com
nexdns zone info example.com          # nameservers to delegate to, SOA, DNSSEC state

# 3. Add records.
nexdns record add example.com A     @      203.0.113.10 --ttl 300
nexdns record add example.com A     www    203.0.113.10
nexdns record add example.com AAAA  @      2001:db8::10
nexdns record add example.com MX    @      mail.example.com --priority 10
nexdns record add example.com TXT   @      "v=spf1 -all"
nexdns record add example.com CAA   @      letsencrypt.org --tag issue --flags 0

# 4. Sign the zone and hand the DS records to your registrar.
nexdns dnssec enable example.com
nexdns dnssec ds-records example.com

# 5. Confirm the world can see it.
nexdns zone check example.com
```

Update the NS records at your registrar to the nameservers `nexdns zone info` prints. Until you do, the zone is live on the NexDNS nameservers but nothing is delegated to it.

## Command reference

Every command accepts the [global flags](#global-flags). `nexdns <group> --help` prints the group; `nexdns <group> <command> --help` prints the flags and examples for one command.

### Authentication and account

| Command | What it does |
| --- | --- |
| `nexdns auth token <TOKEN>` | Verify a token against the API and save it to the config file |
| `nexdns auth status` | Show who you are authenticated as, the API URL, and where the token came from |
| `nexdns auth logout` | Remove the saved token from the config file |
| `nexdns account info` | Account details |
| `nexdns account api-keys` | List the API keys on the account |

API keys carry scopes: `zones.read`, `zones.write`, `records.read`, `records.write`, `webhooks.read`, `webhooks.write`. Give each automation the least-privilege key it needs – a deploy pipeline that only writes records does not need `zones.write`. DNSSEC commands ride on the zone scopes: reading status needs `zones.read`, enabling or disabling needs `zones.write`.

### Zones

| Command | What it does |
| --- | --- |
| `nexdns zone list` | List zones. `--search`, `--page`, `--per-page`, `--all` to walk every page |
| `nexdns zone add <domain>` | Create a zone. `--type master\|slave`, `--master-ip`, `--ns-group` |
| `nexdns zone ensure <domain>` | Create it only if it does not exist – safe to re-run |
| `nexdns zone info <domain>` | Nameservers, SOA, record count, DNSSEC state |
| `nexdns zone export <domain>` | Export as a BIND zone file. `--format bind\|json` |
| `nexdns zone import <domain> <file>` | Import a BIND zone file. `--replace`, `--dry-run` |
| `nexdns zone check <domain>` | Query public resolvers and report what they see |
| `nexdns zone move <domain> <ns-group>` | Move the zone to another nameserver group |
| `nexdns zone delete <domain>` | Delete the zone and its records. `--force` skips the prompt |

```bash
nexdns zone list --all --search example
nexdns zone add example.com --ns-group <slug>
nexdns zone export example.com > example.com.zone
```

Internationalized zones are echoed back in their native spelling rather than as `xn--` punycode, so a zone does not change how it is written between the command that created it and the one that lists it.

**Importing.** `zone import` creates the zone if it is missing, skips records that already match, and reports apex NS records as skipped rather than failing on them – the nameservers of the zone are managed by NexDNS. `--dry-run` prints the plan and changes nothing. `--replace` additionally deletes records that the file does not define, never touching the apex SOA or NS.

```bash
nexdns zone import example.com ./example.com.zone --dry-run
#   + A www 203.0.113.10 (TTL 300)
#   ~ 2 apex NS record(s) skipped – nameservers are managed by NexDNS
#   1 to add, 12 unchanged. Run without --dry-run to apply.
nexdns zone import example.com ./example.com.zone
```

**Checking propagation.** `zone check` asks four public resolvers (Google, Cloudflare, Quad9, OpenDNS) for the zone's NS records, and for the apex A and MX records when the zone has them. It exits non-zero if any check fails, so a migration pipeline can gate on it.

**Moving between nameserver groups.** The zone keeps answering throughout: the new group's nameservers are provisioned before the command returns, and the old ones keep serving while resolvers refresh. Update the delegation at your registrar afterwards to the nameservers `nexdns zone info` prints. The slug is validated against the groups available on your account, and the CLI lists them if it does not match, so a typo is named rather than returned as a generic validation error.

```bash
nexdns zone move example.com <ns-group> --dry-run
nexdns zone move example.com <ns-group>
```

### Records

| Command | What it does |
| --- | --- |
| `nexdns record list <domain>` | List records. `--type`, `--name`, `--search` |
| `nexdns record add <domain> <type> <name> <content>` | Create a record |
| `nexdns record ensure <domain> <type> <name> <content>` | Create it only if an exact type + name + content match is missing |
| `nexdns record update <domain> <record-id>` | Change `--content`, `--record-name`, `--ttl` or `--priority` |
| `nexdns record delete <domain> <record-id>` | Delete a record. `--force` skips the prompt |

Supported types: **A, AAAA, CNAME, MX, TXT, NS, SRV, CAA, PTR, ALIAS, DNAME, DS, TLSA**.

Type-specific values are separate flags rather than hand-composed rdata:

```bash
nexdns record add example.com SRV _sip._tcp sip.example.com \
  --priority 10 --weight 60 --port 5060
nexdns record add example.com CAA @ letsencrypt.org --tag issue --flags 0
nexdns record add example.com TLSA _443._tcp \
  abc123def4567890abc123def4567890abc123def4567890abc123def4567890 \
  --usage 3 --selector 1 --matching-type 1
nexdns record add example.com DS child \
  4d1e2f3a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f6 \
  --keytag 12345 --algorithm 13 --digest-type 2
```

`--ttl` is only sent when you pass it explicitly. A TTL belongs to the whole rrset, so omitting the flag leaves the existing TTL alone instead of silently retiming every other value at that name.

`record ensure` is the idempotent form: it skips when an exact type + name + content match already exists and creates otherwise, so it is safe on round-robin sets, where it will not touch existing records that hold different content.

```bash
# Safe to run on every deploy
nexdns record ensure example.com A app 203.0.113.20
nexdns record ensure example.com A app 203.0.113.21
```

### DNSSEC

| Command | What it does |
| --- | --- |
| `nexdns dnssec status <domain>` | Whether the zone is signed, and its key material |
| `nexdns dnssec enable <domain>` | Sign the zone and print the DS records to publish |
| `nexdns dnssec ds-records <domain>` | Print the DS records again, in any output format |
| `nexdns dnssec disable <domain>` | Unsign the zone. `--force` skips the prompt |

Keys are generated and the zone is signed on the NexDNS nameservers – there is nothing to install or hold locally. What you do have to do is publish the DS records at your registrar, which is what closes the chain of trust.

```bash
nexdns dnssec enable example.com
# DNSSEC enabled for example.com
#
# DS Records (add these at your registrar):
#   <one line per DS record, ready to paste>


# Machine-readable, for a registrar API
nexdns dnssec ds-records example.com --output json
```

Remove the DS records at the registrar and wait for the old ones to expire from resolver caches **before** running `dnssec disable`, or resolvers that still have the DS cached will treat the now-unsigned zone as broken.

### Webhooks

Subscribe one of your endpoints to the events your zones emit. The signing secret is printed once, on creation, and is what your receiver uses to verify each delivery.

| Command | What it does |
| --- | --- |
| `nexdns webhook list` | List the subscriptions on the account |
| `nexdns webhook show <webhook-id>` | One subscription plus its recent delivery attempts |
| `nexdns webhook create <url>` | Subscribe an endpoint. `--events` (required), `--description` |
| `nexdns webhook update <webhook-id>` | Change `--url`, `--events`, `--description` or `--active` |
| `nexdns webhook test <webhook-id>` | Queue a test event for delivery |
| `nexdns webhook delete <webhook-id>` | Remove the subscription. `--force` skips the prompt |

```bash
nexdns webhook create https://example.com/hooks/dns \
  --events zone.created,zone.deleted,record.created \
  --description "production"
# Webhook created (ID: xK9mPq2R)
# Signing secret: whk_ExampleSigningSecretNotARealOne1234567890
# Store the secret now – it is not shown again.

# Queue a test event, then read the delivery attempts back
nexdns webhook test xK9mPq2R
nexdns webhook show xK9mPq2R

# Pause deliveries without deleting the endpoint
nexdns webhook update xK9mPq2R --active=false
```

Event types:

| | | |
| --- | --- | --- |
| `zone.created` | `zone.updated` | `zone.deleted` |
| `record.created` | `record.updated` | `record.deleted` |
| `dnssec.enabled` | `dnssec.disabled` | |
| `zone.health.problem` | `zone.health.resolved` | |

`webhook test` queues the event rather than delivering it synchronously: a success means the platform accepted it, and `webhook show` is where you read the outcome. Requires an API key with the `webhooks.read` / `webhooks.write` scopes.

## DNS-as-Code

Declare zones in YAML, keep the file in git, and let CI reconcile the account against it. `nexdns apply` is a dry run by default and only writes with `--confirm`.

```yaml
# nexdns.yaml
zones:
  example.com:
    dnssec: true
    records:
      - { type: A,     name: "@",      content: "203.0.113.10", ttl: 300 }
      - { type: A,     name: www,      content: "203.0.113.10" }
      - { type: AAAA,  name: "@",      content: "2001:db8::10" }
      - { type: CNAME, name: api,      content: "example.com" }
      - { type: MX,    name: "@",      content: "mail.example.com", priority: 10 }
      - { type: TXT,   name: "@",      content: "v=spf1 include:_spf.example.com -all" }
      - { type: TXT,   name: _dmarc,   content: "v=DMARC1; p=reject; rua=mailto:dmarc@example.com" }
      - { type: SRV,   name: _sip._tcp, content: "sip.example.com", priority: 10, weight: 60, port: 5060 }
      - { type: CAA,   name: "@",      content: "letsencrypt.org", tag: issue, flags: 0 }
```

```bash
nexdns diff                    # what would change
nexdns apply                   # same thing – apply is a dry run until you confirm
nexdns apply --confirm         # write it
nexdns apply --confirm --delete  # also remove records the file does not define
nexdns apply --confirm --zone example.com   # one zone out of a multi-zone file
nexdns pull example.com > nexdns.yaml       # generate the file from what is live
```

| Flag | Applies to | Meaning |
| --- | --- | --- |
| `--file` | `apply`, `diff`, `pull` | Config file path (`apply` and `diff` default to `nexdns.yaml`) |
| `--confirm` | `apply` | Actually write the changes |
| `--zone` | `apply`, `diff` | Restrict to one zone defined in the file |
| `--delete` | `apply`, `diff` | Remove remote records the file does not define |
| `--append` | `pull` | Append to `--file` instead of overwriting |

**Behaviour worth knowing before you put this in CI**

- A zone named in the file but absent on the account is **created** by `apply --confirm`. `dnssec: true` enables signing on a zone that is not yet signed; it is not turned off by setting it back to `false`.
- `pull` omits the apex SOA and NS records, because they are managed for you – so a `pull` followed by an `apply` is a no-op rather than a fight over the zone's own nameservers.
- `--zone` naming something the file does not define is an **error**, not an empty run. A typo in a pipeline should not look like a converged apply.
- `${VAR}` references anywhere in the file are substituted from the environment, and a reference with no value **fails the parse** instead of publishing the literal `${VAR}` as a record value.
- `apply --confirm` exits non-zero if any operation failed, after attempting them all and printing which ones failed.

```yaml
# Same file, different environments
zones:
  ${DOMAIN}:
    records:
      - { type: A, name: "@",   content: "${SERVER_IP}" }
      - { type: A, name: www,   content: "${SERVER_IP}" }
```

```yaml
# .github/workflows/dns.yml
- name: Plan
  run: nexdns diff
  env:
    NEXDNS_TOKEN: ${{ secrets.NEXDNS_TOKEN }}

- name: Apply
  if: github.ref == 'refs/heads/main'
  run: nexdns apply --confirm
  env:
    NEXDNS_TOKEN: ${{ secrets.NEXDNS_TOKEN }}
```

## ACME / certificates

`nexdns acme hook` writes and removes the `_acme-challenge` TXT record for a DNS-01 challenge. It finds the right zone by walking up from the name being validated, so a challenge for `sub.example.com` lands in the `example.com` zone as `_acme-challenge.sub`.

It reads `CERTBOT_DOMAIN` and `CERTBOT_VALIDATION` from the environment, which is exactly what certbot's manual hooks set:

```bash
certbot certonly --manual \
  --preferred-challenges dns \
  --manual-auth-hook "nexdns acme hook --action create" \
  --manual-cleanup-hook "nexdns acme hook --action delete" \
  -d '*.example.com' -d example.com
```

Or drive it directly, from any ACME client that can shell out:

```bash
nexdns acme hook --action create --domain example.com --token "<validation-token>"
nexdns acme hook --action delete --domain example.com --token "<validation-token>"
```

For certbot there is also a native plugin that needs no shell hooks – see [certbot-dns-nexdns](https://github.com/nexdns/certbot-dns-nexdns). Full walkthrough at [nexdns.tech/docs/acme](https://nexdns.tech/docs/acme).

Issue the key this runs with with `records.read` + `records.write` only. An ACME hook has no business being able to delete a zone.

## Output formats and scripting

Commands that return data honour `--output` / `-o`: `table` (the default), `json`, `yaml`, `csv`. That covers `zone list`, `zone info`, `zone move`, `record list`, `dnssec status`, `dnssec ds-records`, `account info`, `account api-keys`, `webhook list`, `webhook show` and `webhook update`. The commands that report progress (`apply`, `zone import`, `zone check`, `record add`) print plain text, and `zone export` prints the zone file itself.

```bash
nexdns zone list --all -o json | jq -r '.[].name'
nexdns record list example.com -o csv > records.csv
nexdns zone info example.com -o yaml
```

**Exit codes**

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | The operation failed – API error, network failure, a failed propagation check |
| `2` | Usage error – unknown command or flag, wrong number of arguments |

A misspelled subcommand fails with code `2` rather than printing help and succeeding, so a typo in a pipeline is a scriptable error and not a silent no-op.

```bash
# Bulk-retime every A record in a zone
nexdns record list example.com -o json \
  | jq -r '.[] | select(.type == "A") | .id' \
  | xargs -I{} nexdns record update example.com {} --ttl 300

# Gate a deploy on the zone actually resolving
if ! nexdns zone check example.com; then
  echo "DNS not propagated yet" >&2
  exit 1
fi

# Report every zone that is not signed
nexdns zone list --all -o json | jq -r '.[].name' | while read -r zone; do
  nexdns dnssec status "$zone" -o json | jq -e '.enabled' >/dev/null || echo "unsigned: $zone"
done
```

`--quiet` suppresses non-essential output, and colour follows `--color auto|always|never` and the [`NO_COLOR`](https://no-color.org) convention.

## Configuration

Settings resolve in increasing priority: **built-in default → config file → environment variable → explicitly set flag.**

### Environment variables

| Variable | Purpose |
| --- | --- |
| `NEXDNS_TOKEN` | API token. The usual choice in CI |
| `NEXDNS_API_URL` | API base URL |
| `NEXDNS_CONFIG` | Config file path |
| `NEXDNS_TIMEOUT` | HTTP timeout in seconds |
| `NO_COLOR` | Present at any value – disables colour |
| `HTTPS_PROXY`, `HTTP_PROXY` | Honoured by Go's HTTP client, no extra configuration |

### Config file

`~/.nexdns/config.yaml`, written with owner-only permissions because it holds the token. `nexdns auth token` creates it; `nexdns config` edits it.

```bash
nexdns config view                    # resolved settings, token masked
nexdns config set output json         # keys: api-url, token, output, color, timeout
nexdns config get api-url
```

```yaml
# ~/.nexdns/config.yaml
api_url: https://api.nexdns.tech/v1
token: nxd_exampletoken1234567890
output: table
color: auto
timeout: 30
```

### Global flags

| Flag | Default | Description |
| --- | --- | --- |
| `--token` | (none) | API token, overriding config file and environment |
| `--api-url` | `https://api.nexdns.tech/v1` | API base URL |
| `--config` | (none) | Config file path (default `~/.nexdns/config.yaml`) |
| `--output`, `-o` | `table` | Output format: `table`, `json`, `yaml`, `csv` |
| `--color` | `auto` | Colour mode: `auto`, `always`, `never` |
| `--quiet`, `-q` | `false` | Suppress non-essential output |
| `--verbose`, `-v` | `false` | Print each HTTP request, its status, and the rate-limit budget |
| `--dry-run` | `false` | Preview the change without applying it |
| `--timeout` | `30` | HTTP timeout in seconds |

`--dry-run` previews without writing on `zone add`, `zone ensure`, `zone delete`, `zone move`, `zone import`, `record add`, `record ensure`, `record update`, `record delete`, `dnssec enable`, `dnssec disable`, and every `webhook` command that changes something. `apply` has its own form of this: it is a dry run until you pass `--confirm`.

## Migrating from another provider

**BIND zone import / export** – migrate from any provider that speaks BIND zone-file format (Route 53, Cloudflare, DNSmanager, PowerDNS, BIND, NSD). The path is the same whichever one you are leaving: get a zone file out of it, import, verify, repoint.

```bash
# 1. Import (dry run first – it prints exactly what it would create)
nexdns zone import example.com ./example.com.zone --dry-run
nexdns zone import example.com ./example.com.zone

# 2. Compare against the old provider's answers, then look at the delegation
nexdns record list example.com
nexdns zone info example.com          # the nameservers to set at your registrar

# 3. After repointing, confirm public resolvers agree
nexdns zone check example.com
```

**Getting the zone file out**

| Leaving | How |
| --- | --- |
| Cloudflare | Export the zone file from the dashboard's DNS section |
| AWS Route 53 | `cli53 export example.com > example.com.zone` |
| ISPmanager / DNSmanager | Export the domain as a zone file, or point the [ISPmanager-compatible API](https://nexdns.tech/docs/integrations) at NexDNS instead |
| A self-hosted nameserver | The zone files you already have |

**Keeping the old provider live while you cut over.** Create the zone as a secondary and let it transfer from your existing primary, so the NexDNS nameservers answer identically to the ones you are leaving before you repoint the delegation:

```bash
nexdns zone add example.com --type slave --master-ip 203.0.113.53
```

**Lowering TTLs first.** A day before the cutover, drop the TTL on the records that matter so resolvers pick up the change quickly:

```bash
nexdns record list example.com -o json \
  | jq -r '.[] | select(.type == "A" or .type == "MX") | .id' \
  | xargs -I{} nexdns record update example.com {} --ttl 300
```

Longer form, with the registrar-side steps, at [nexdns.tech/docs/zones](https://nexdns.tech/docs/zones).

## Troubleshooting

**`not authenticated. Run nexdns auth token <TOKEN> to authenticate`**
No token was found in the flag, the environment, or the config file. `nexdns auth status` reports which of the three a token is coming from when there is one.

**`invalid token format: must start with 'nxd_' prefix`**
That is not an API key. Create one at [nexdns.tech/settings/api-keys](https://nexdns.tech/settings/api-keys).

**`token validation failed`**
The token was rejected by the API it was sent to. Check `nexdns auth status` for the API URL in effect, and that the key has not been revoked or expired.

**`zone "example.com" not found`**
The zone is not on this account, or the key lacks `zones.read`. `nexdns zone list --all --search example` shows what the key can actually see.

**A 403 on a command that used to work**
The key is missing a scope. Writes need `zones.write` / `records.write`; webhooks need their own pair. `nexdns account api-keys` lists the keys and what each one carries.

**Rate limiting on a bulk import**
Handled for you: the client reads the rate-limit budget from each response and waits for a short window to roll over rather than spending attempts on rejections, so a large import slows down instead of losing records. A window too long to wait out is reported with the reason and how long is left, rather than hanging.

**Transient network or 5xx failures**
Read requests are retried, up to three attempts in total, with exponential backoff and jitter. Writes are not retried automatically – replaying a create that may already have been applied would double-submit – so a failed write is reported for you to decide about.

**Seeing what is actually on the wire**
`--verbose` prints each request, its status, the duration, and the rate-limit headers. Nothing else changes.

```bash
nexdns zone list --verbose
```

**Corporate proxy**
Set `HTTPS_PROXY` / `HTTP_PROXY`; Go's HTTP client picks them up with no further configuration.

**Windows**
Download the `windows` archive from the [releases page](https://github.com/nexdns/cli/releases), or `go install`. Everything in this README works the same, with `%USERPROFILE%\.nexdns\config.yaml` as the config path.

## Related projects

| Project | Use it when |
| --- | --- |
| [Terraform provider](https://github.com/nexdns/terraform-provider-nexdns) | DNS is part of a larger infrastructure state you already manage in Terraform |
| [OctoDNS provider](https://github.com/nexdns/octodns-nexdns) | You manage several DNS providers at once and want one YAML source for all of them |
| [Certbot plugin](https://github.com/nexdns/certbot-dns-nexdns) | Certbot issues your certificates and you want a native DNS-01 plugin instead of shell hooks |
| [Homebrew tap](https://github.com/nexdns/homebrew-tap) | Installing and updating the CLI on macOS or Linux |

`nexdns apply` is the lighter-weight option when DNS is managed on its own: one binary, one YAML file, no state file to keep.

### Other DNS command-line tools

Each of these drives its own platform, and they are listed so you can tell at a glance which tool belongs to which:

| Tool | Manages |
| --- | --- |
| **`nexdns`** | Zones and records on a NexDNS account |
| `cli53` | AWS Route 53 |
| `aws route53` | AWS Route 53 |
| `flarectl` | Cloudflare |
| `doctl compute dns` | DigitalOcean |
| `pdnsutil` | Self-hosted PowerDNS |
| [octodns-nexdns](https://github.com/nexdns/octodns-nexdns) | NexDNS through OctoDNS, alongside other providers |

## Links

- **Website**: [nexdns.tech](https://nexdns.tech)
- **CLI documentation**: [nexdns.tech/docs/cli](https://nexdns.tech/docs/cli)
- **API reference**: [nexdns.tech/docs/api](https://nexdns.tech/docs/api)
- **Integrations**: [nexdns.tech/docs/integrations](https://nexdns.tech/docs/integrations)
- **Pricing**: [nexdns.tech/pricing](https://nexdns.tech/pricing)
- **Issues and feature requests**: [GitHub Issues](https://github.com/nexdns/cli/issues)
- **Changelog**: [CHANGELOG.md](CHANGELOG.md)

Found a security issue? Please do not open a public issue – report it through [nexdns.tech/contact](https://nexdns.tech/contact) so it can be fixed before it is public.

## License

[Apache-2.0](LICENSE)
