package dns

// ttlPtr converts a parsed TTL into the API request's optional field. Zero means
// the zone file did not carry one, which is where the API keeps the TTL an
// existing rrset already has.
func ttlPtr(ttl int) *int {
	if ttl == 0 {
		return nil
	}
	return &ttl
}
