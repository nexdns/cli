package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

const lookupTimeout = 5 * time.Second

func resolverDialer(resolver string) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: lookupTimeout}
		return d.DialContext(ctx, "udp", resolver+":53")
	}
}

func newResolver(server string) *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial:     resolverDialer(server),
	}
}

// LookupNS queries NS records for a domain using a specific resolver.
func LookupNS(domain, resolver string) ([]string, error) {
	r := newResolver(resolver)
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	nss, err := r.LookupNS(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("lookup failed: %w", err)
	}

	var results []string
	for _, ns := range nss {
		results = append(results, strings.TrimSuffix(ns.Host, "."))
	}
	return results, nil
}

// LookupA queries A records for a domain using a specific resolver.
func LookupA(domain, resolver string) ([]string, error) {
	r := newResolver(resolver)
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	ips, err := r.LookupIPAddr(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("lookup failed: %w", err)
	}

	var results []string
	for _, ip := range ips {
		if ip.IP.To4() != nil {
			results = append(results, ip.IP.String())
		}
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no A records found")
	}
	return results, nil
}

// LookupMX queries MX records for a domain using a specific resolver.
func LookupMX(domain, resolver string) ([]string, error) {
	r := newResolver(resolver)
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	mxs, err := r.LookupMX(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("lookup failed: %w", err)
	}

	var results []string
	for _, mx := range mxs {
		results = append(results, fmt.Sprintf("%d %s", mx.Pref, strings.TrimSuffix(mx.Host, ".")))
	}
	return results, nil
}
