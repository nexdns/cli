package dns

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/nexdns/cli/internal/api"
)

// ParsedRecord represents a DNS record parsed from a BIND zone file.
type ParsedRecord struct {
	Name     string
	TTL      int
	Type     string
	Content  string
	Priority *int
	Weight   *int
	Port     *int
	Flags    *int
	Tag      string
}

// ParseBINDZone parses a BIND zone file and returns a list of records.
// Supports $TTL and $ORIGIN directives, comments, multi-line records in parentheses.
func ParseBINDZone(r io.Reader, origin string) ([]ParsedRecord, error) {
	if origin != "" && !strings.HasSuffix(origin, ".") {
		origin += "."
	}

	scanner := bufio.NewScanner(r)
	var records []ParsedRecord
	defaultTTL := 3600
	currentOrigin := origin

	var multiLine strings.Builder
	inParens := false
	// A line that starts with whitespace inherits the previous owner name
	// (RFC 1035 section 5.1).
	lastOwner := "@"

	for scanner.Scan() {
		line := scanner.Text()

		// Strip comments (but not inside quoted strings)
		line = stripComment(line)
		ownerInherited := line != "" && (line[0] == ' ' || line[0] == '\t')
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		// Handle multi-line records with parentheses
		if inParens {
			multiLine.WriteString(" ")
			multiLine.WriteString(line)
			if strings.Contains(line, ")") {
				inParens = false
				line = multiLine.String()
				line = strings.ReplaceAll(line, "(", "")
				line = strings.ReplaceAll(line, ")", "")
				line = strings.TrimSpace(line)
			} else {
				continue
			}
		} else if strings.Contains(line, "(") && !strings.Contains(line, ")") {
			inParens = true
			multiLine.Reset()
			multiLine.WriteString(line)
			continue
		} else {
			// Single-line: remove any parentheses
			line = strings.ReplaceAll(line, "(", "")
			line = strings.ReplaceAll(line, ")", "")
		}

		// Handle directives
		if strings.HasPrefix(line, "$TTL") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if ttl, err := parseTTLValue(parts[1]); err == nil {
					defaultTTL = ttl
				}
			}
			continue
		}
		if strings.HasPrefix(line, "$ORIGIN") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentOrigin = parts[1]
			}
			continue
		}
		if strings.HasPrefix(line, "$") {
			// Skip unknown directives
			continue
		}

		record, owner, err := parseRecordLine(line, defaultTTL, currentOrigin, ownerInherited, lastOwner)
		if err != nil {
			continue // skip unparseable lines
		}
		if owner != "" {
			lastOwner = owner
		}
		if record != nil {
			records = append(records, *record)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading zone file: %w", err)
	}

	return records, nil
}

// ToCreateRequests converts parsed records to API create requests.
func ToCreateRequests(records []ParsedRecord) []api.CreateRecordRequest {
	var reqs []api.CreateRecordRequest
	for _, r := range records {
		req := api.CreateRecordRequest{
			Name:     r.Name,
			Type:     r.Type,
			Content:  r.Content,
			TTL:      api.OptionalTTL(r.TTL),
			Priority: r.Priority,
			Weight:   r.Weight,
			Port:     r.Port,
			Flags:    r.Flags,
			Tag:      r.Tag,
		}
		reqs = append(reqs, req)
	}
	return reqs
}

// parseRecordLine parses one record line. ownerInherited reports whether the
// line began with whitespace, in which case the owner name is inherited from
// lastOwner. It returns the record (nil for a skipped type), the owner name to
// carry to the next line, and a parse error.
func parseRecordLine(line string, defaultTTL int, origin string, ownerInherited bool, lastOwner string) (*ParsedRecord, string, error) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return nil, "", fmt.Errorf("too few fields")
	}

	// Parse: [name] [ttl] [class] type rdata...
	idx := 0
	name := ""
	ttl := defaultTTL

	inheritedName := "@"
	if ownerInherited && lastOwner != "" {
		inheritedName = lastOwner
	}

	// The first field is the owner name, unless the line began with whitespace:
	// then it may be a TTL, class or type. Labels such as "ns" or "mx" are
	// valid owner names, so the token alone does not decide.
	switch {
	case !ownerInherited:
		name = fields[idx]
		idx++
	case isRecordType(fields[idx]):
		// No name, TTL or class: starts with the type
		name = inheritedName
	case isClass(fields[idx]):
		// No name: starts with the class
		name = inheritedName
	case isTTL(fields[idx]):
		// No name: starts with the TTL
		name = inheritedName
		if t, err := parseTTLValue(fields[idx]); err == nil {
			ttl = t
		}
		idx++
	default:
		name = fields[idx]
		idx++
	}

	if idx >= len(fields) {
		return nil, "", fmt.Errorf("unexpected end of line")
	}

	// Optional TTL
	if idx < len(fields) && isTTL(fields[idx]) && !isRecordType(fields[idx]) && !isClass(fields[idx]) {
		if t, err := parseTTLValue(fields[idx]); err == nil {
			ttl = t
		}
		idx++
	}

	// Optional class (IN, CH, HS)
	if idx < len(fields) && isClass(fields[idx]) {
		idx++
	}

	if idx >= len(fields) {
		return nil, "", fmt.Errorf("no record type found")
	}

	// Record type
	recordType := strings.ToUpper(fields[idx])
	idx++

	if idx >= len(fields) {
		return nil, "", fmt.Errorf("no rdata found")
	}

	// Normalize name
	name = normalizeName(name, origin)

	// Skip SOA (the platform owns it), but the owner still carries forward.
	if recordType == "SOA" {
		return nil, name, nil
	}

	// Parse rdata based on type
	rdata := fields[idx:]
	record, err := parseRData(name, ttl, recordType, rdata, origin)
	if err != nil {
		return nil, name, err
	}
	return record, name, nil
}

func parseRData(name string, ttl int, recordType string, rdata []string, origin string) (*ParsedRecord, error) {
	record := &ParsedRecord{
		Name: name,
		TTL:  ttl,
		Type: recordType,
	}

	switch recordType {
	case "A", "AAAA":
		if len(rdata) < 1 {
			return nil, fmt.Errorf("missing address")
		}
		record.Content = rdata[0]

	case "CNAME", "NS", "PTR", "ALIAS", "DNAME":
		if len(rdata) < 1 {
			return nil, fmt.Errorf("missing hostname")
		}
		record.Content = absoluteHostname(rdata[0], origin)

	case "MX":
		if len(rdata) < 2 {
			return nil, fmt.Errorf("MX needs priority and host")
		}
		priority, err := strconv.Atoi(rdata[0])
		if err != nil {
			return nil, fmt.Errorf("invalid MX priority: %s", rdata[0])
		}
		record.Priority = &priority
		record.Content = absoluteHostname(rdata[1], origin)

	case "TXT":
		// Keep TXT quoting as written: several character-strings ("a" "b") must
		// stay separate.
		txt := strings.Join(rdata, " ")
		if !strings.HasPrefix(txt, `"`) || !strings.HasSuffix(txt, `"`) {
			txt = strings.ReplaceAll(txt, `"`, "")
		}
		record.Content = txt

	case "SRV":
		// priority weight port target
		if len(rdata) < 4 {
			return nil, fmt.Errorf("SRV needs priority, weight, port, target")
		}
		priority, _ := strconv.Atoi(rdata[0])
		weight, _ := strconv.Atoi(rdata[1])
		port, _ := strconv.Atoi(rdata[2])
		record.Priority = &priority
		record.Weight = &weight
		record.Port = &port
		record.Content = absoluteHostname(rdata[3], origin)

	case "CAA":
		// flags tag value
		if len(rdata) < 3 {
			return nil, fmt.Errorf("CAA needs flags, tag, value")
		}
		flags, _ := strconv.Atoi(rdata[0])
		record.Flags = &flags
		record.Tag = rdata[1]
		record.Content = strings.Trim(rdata[2], `"`)

	case "DS":
		// keytag algorithm digest_type digest
		if len(rdata) < 4 {
			return nil, fmt.Errorf("DS needs keytag, algorithm, digest_type, digest")
		}
		record.Content = strings.Join(rdata, " ")

	case "TLSA":
		// usage selector matching_type certificate
		if len(rdata) < 4 {
			return nil, fmt.Errorf("TLSA needs usage, selector, matching_type, certificate")
		}
		record.Content = strings.Join(rdata, " ")

	default:
		// Generic: join all rdata as content
		record.Content = strings.Join(rdata, " ")
	}

	return record, nil
}

func normalizeName(name, origin string) string {
	if name == "@" || name == "" {
		return "@"
	}
	// Remove trailing dot
	name = strings.TrimSuffix(name, ".")
	// If name equals origin (without dot), it's the apex
	originClean := strings.TrimSuffix(origin, ".")
	if name == originClean {
		return "@"
	}
	// If name ends with .origin, extract relative part
	if originClean != "" && strings.HasSuffix(name, "."+originClean) {
		return strings.TrimSuffix(name, "."+originClean)
	}
	return name
}

// absoluteHostname returns an rdata hostname fully qualified, without the
// trailing dot. A relative name is expanded with $ORIGIN.
func absoluteHostname(hostname, origin string) string {
	originClean := strings.TrimSuffix(origin, ".")

	if strings.HasSuffix(hostname, ".") {
		return strings.TrimSuffix(hostname, ".")
	}
	// "." is the MX null target / SRV "no service" sentinel.
	if hostname == "" || hostname == "@" {
		return originClean
	}
	if originClean == "" {
		return hostname
	}
	return hostname + "." + originClean
}

func stripComment(line string) string {
	inQuote := false
	for i, ch := range line {
		if ch == '"' {
			inQuote = !inQuote
		}
		if ch == ';' && !inQuote {
			return line[:i]
		}
	}
	return line
}

func isRecordType(s string) bool {
	types := map[string]bool{
		"A": true, "AAAA": true, "CNAME": true, "MX": true, "NS": true,
		"TXT": true, "SRV": true, "PTR": true, "CAA": true, "SOA": true,
		"DS": true, "TLSA": true, "ALIAS": true, "DNAME": true,
	}
	return types[strings.ToUpper(s)]
}

// isClass reports whether the token is a DNS class. Like types, classes are
// case-insensitive (RFC 1035 section 2.3.1).
func isClass(s string) bool {
	switch strings.ToUpper(s) {
	case "IN", "CH", "HS":
		return true
	default:
		return false
	}
}

func isTTL(s string) bool {
	// Pure number or number with time suffix (h, m, s, d, w)
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	if _, err := parseTTLValue(s); err == nil {
		return true
	}
	return false
}

func parseTTLValue(s string) (int, error) {
	// Try plain number first
	if v, err := strconv.Atoi(s); err == nil {
		return v, nil
	}

	// Try time suffixes: 1h, 30m, 1d, 1w, etc.
	s = strings.ToLower(s)
	total := 0
	current := ""
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			current += string(ch)
		} else {
			if current == "" {
				return 0, fmt.Errorf("invalid TTL: %s", s)
			}
			num, _ := strconv.Atoi(current)
			switch ch {
			case 'w':
				total += num * 604800
			case 'd':
				total += num * 86400
			case 'h':
				total += num * 3600
			case 'm':
				total += num * 60
			case 's':
				total += num
			default:
				return 0, fmt.Errorf("invalid TTL suffix: %c", ch)
			}
			current = ""
		}
	}
	if current != "" {
		// Trailing number without suffix: seconds
		num, _ := strconv.Atoi(current)
		total += num
	}
	if total == 0 {
		return 0, fmt.Errorf("invalid TTL: %s", s)
	}
	return total, nil
}
