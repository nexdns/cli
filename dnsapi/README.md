# acme.sh DNS hook for NexDNS

`dns_nexdns.sh` is the DNS-01 hook for [acme.sh](https://acme.sh). It is
submitted for inclusion in acme.sh itself; until a release carries it, copy it
in by hand:

```bash
cp dnsapi/dns_nexdns.sh ~/.acme.sh/dnsapi/
```

Then issue a certificate the usual way:

```bash
export NEXDNS_Token="nxd_your_api_token"

acme.sh --issue --dns dns_nexdns -d example.com -d '*.example.com'
```

The token needs the `zones.read`, `records.read` and `records.write`
permissions, and is created under
[account settings](https://nexdns.tech/settings/api-keys). `NEXDNS_Api`
overrides the API base URL and defaults to `https://api.nexdns.tech/v1`.

acme.sh saves both values on first use and reuses them on renewal, so a cron
renewal needs no environment of its own.

## Where to report a problem

The header of the script points at the acme.sh issue tracker, which is where it
will belong once acme.sh ships the hook. Until then it is ours: report anything
wrong with it through [NexDNS support](https://nexdns.tech/support) rather than
to the acme.sh maintainers, who have not released this file.

## Full guide

[nexdns.tech/docs/acme](https://nexdns.tech/docs/acme) covers the same ground
for certbot, lego and Caddy as well.
