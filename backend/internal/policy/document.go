package policy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
)

const RelayCapability = "tailscale.com/cap/relay"

type Document struct {
	Raw    string
	Root   map[string]json.RawMessage
	Grants []json.RawMessage
	Groups map[string][]string
	Hosts  map[string]string
}

type Grant struct {
	Src        []string
	Dst        []string
	PeerRelay  bool
	ExactShape bool
}

func Parse(raw string) (*Document, error) {
	clean, err := standardJSON([]byte(raw))
	if err != nil {
		return nil, err
	}
	if err := rejectDuplicateKeys(clean); err != nil {
		return nil, err
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(clean, &root); err != nil || root == nil {
		return nil, errors.New("Headscale Policy root must be an object")
	}
	doc := &Document{Raw: raw, Root: root, Groups: map[string][]string{}, Hosts: map[string]string{}}
	if value, ok := root["grants"]; ok {
		if err := json.Unmarshal(value, &doc.Grants); err != nil || doc.Grants == nil {
			return nil, errors.New("Headscale Policy grants must be an array")
		}
	}
	if value, ok := root["groups"]; ok {
		if err := json.Unmarshal(value, &doc.Groups); err != nil {
			return nil, errors.New("Headscale Policy groups must be an object")
		}
	}
	if value, ok := root["hosts"]; ok {
		if err := json.Unmarshal(value, &doc.Hosts); err != nil {
			return nil, errors.New("Headscale Policy hosts must be an object")
		}
	}
	return doc, nil
}

func (d *Document) Hash() string {
	clean, err := standardJSON([]byte(d.Raw))
	if err != nil {
		return ""
	}
	var value any
	if json.Unmarshal(clean, &value) != nil {
		return ""
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

func (d *Document) HasGrants() bool {
	_, ok := d.Root["grants"]
	return ok
}

func (d *Document) ReplaceGrants(grants []json.RawMessage) (string, error) {
	if !d.HasGrants() {
		return "", errors.New("the current Headscale Policy has no grants array; refusing to add one because that can change the tailnet's default access")
	}
	encoded, err := json.MarshalIndent(grants, "", "  ")
	if err != nil {
		return "", err
	}
	start, end, found, err := findRootFieldValue([]byte(d.Raw), "grants")
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.New("could not locate the grants array in the HuJSON policy")
	}
	mutated := make([]byte, 0, len(d.Raw)+len(encoded))
	mutated = append(mutated, d.Raw[:start]...)
	mutated = append(mutated, encoded...)
	mutated = append(mutated, d.Raw[end:]...)
	if _, err := Parse(string(mutated)); err != nil {
		return "", fmt.Errorf("updated policy is invalid: %w", err)
	}
	return string(mutated), nil
}

func (d *Document) PeerRelayGrants() ([]Grant, error) {
	result := make([]Grant, 0)
	for _, raw := range d.Grants {
		grant, err := parseGrant(raw)
		if err != nil {
			return nil, err
		}
		if grant.PeerRelay {
			result = append(result, grant)
		}
	}
	return result, nil
}

func parseGrant(raw json.RawMessage) (Grant, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return Grant{}, errors.New("Policy contains an invalid grant object")
	}
	var grant Grant
	if err := json.Unmarshal(fields["src"], &grant.Src); err != nil {
		return Grant{}, errors.New("Policy grant src must be an array")
	}
	if err := json.Unmarshal(fields["dst"], &grant.Dst); err != nil {
		return Grant{}, errors.New("Policy grant dst must be an array")
	}
	app := map[string]json.RawMessage{}
	if rawApp, ok := fields["app"]; ok {
		if err := json.Unmarshal(rawApp, &app); err != nil || app == nil {
			return Grant{}, errors.New("Policy grant app must be an object")
		}
	}
	if capability, ok := app[RelayCapability]; ok {
		var args []json.RawMessage
		if err := json.Unmarshal(capability, &args); err != nil {
			return Grant{}, errors.New("Peer Relay capability value must be an array")
		}
		grant.PeerRelay = true
	}
	grant.ExactShape = len(fields) == 3 && len(grant.Src) == 1 && len(grant.Dst) == 1 && grant.PeerRelay && len(app) == 1
	return grant, nil
}

func ParseGrant(raw json.RawMessage) (Grant, error) { return parseGrant(raw) }

func NewRelayGrant(source, relay string) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"src": []string{source},
		"dst": []string{relay},
		"app": map[string]any{RelayCapability: []any{}},
	})
}

func (d *Document) ExactRelayGrantCount(source, relay string) (int, error) {
	count := 0
	for _, raw := range d.Grants {
		grant, err := parseGrant(raw)
		if err != nil {
			return 0, err
		}
		if grant.ExactShape && grant.Src[0] == source && grant.Dst[0] == relay {
			count++
		}
	}
	return count, nil
}

func (d *Document) EffectiveRelay(source, relay DeviceSelector) (exact, inherited, unknown bool, err error) {
	for _, raw := range d.Grants {
		grant, parseErr := parseGrant(raw)
		if parseErr != nil {
			return false, false, false, parseErr
		}
		if !grant.PeerRelay {
			continue
		}
		sourceMatch, sourceUnknown := selectorsMatch(grant.Src, source, relay, d.Groups, d.Hosts, false)
		relayMatch, relayUnknown := selectorsMatch(grant.Dst, relay, source, d.Groups, d.Hosts, true)
		if sourceMatch && relayMatch {
			if grant.ExactShape && selectorMatchesIP(grant.Src[0], source.IPs) && selectorMatchesIP(grant.Dst[0], relay.IPs) {
				exact = true
			} else {
				inherited = true
			}
		} else if sourceUnknown || relayUnknown {
			unknown = true
		}
	}
	return exact, inherited, unknown, nil
}

type DeviceSelector struct {
	UserID    string
	Username  string
	Email     string
	Hostname  string
	GivenName string
	Tags      []string
	IPs       []string
	Tagged    bool
}

func selectorsMatch(selectors []string, device, peer DeviceSelector, groups map[string][]string, hosts map[string]string, allowSelf bool) (bool, bool) {
	matched, unknown := false, false
	for _, selector := range selectors {
		contains, known := selectorMatches(selector, device, peer, groups, hosts, allowSelf, 0)
		matched = matched || contains
		unknown = unknown || !known
	}
	return matched, unknown
}

var wildcardUser = regexp.MustCompile(`^\*@(.+)$`)

func selectorMatches(selector string, device, peer DeviceSelector, groups map[string][]string, hosts map[string]string, allowSelf bool, depth int) (bool, bool) {
	selector = strings.TrimSpace(selector)
	if depth > 8 || selector == "" {
		return false, false
	}
	if selector == "*" {
		return true, true
	}
	if strings.HasPrefix(selector, "group:") {
		members, ok := groups[selector]
		if !ok {
			return false, false
		}
		for _, member := range members {
			if selectorGroupMember(member, device.Username, device.Email) {
				return true, true
			}
			if strings.HasPrefix(member, "group:") {
				if match, known := selectorMatches(member, device, peer, groups, hosts, allowSelf, depth+1); match || !known {
					return match, known
				}
			}
		}
		return false, true
	}
	if strings.HasPrefix(selector, "tag:") {
		for _, tag := range device.Tags {
			if tag == selector {
				return true, true
			}
		}
		return false, true
	}
	if selector == "autogroup:tagged" {
		return device.Tagged, true
	}
	if selector == "autogroup:member" {
		return !device.Tagged, true
	}
	if selector == "autogroup:self" {
		if !allowSelf || device.UserID == "" || peer.UserID == "" {
			return false, false
		}
		return device.UserID == peer.UserID, true
	}
	if value, ok := hosts[selector]; ok {
		return selectorMatchesIP(value, device.IPs), true
	}
	if match, known := selectorMatchesIPValue(selector, device.IPs); known {
		return match, true
	}
	if match := wildcardUser.FindStringSubmatch(selector); len(match) == 2 {
		return strings.HasSuffix(strings.ToLower(device.Username), "@"+strings.ToLower(match[1])), true
	}
	if strings.HasSuffix(selector, "@") {
		return strings.EqualFold(strings.TrimSuffix(device.Username, "@"), strings.TrimSuffix(selector, "@")), true
	}
	if strings.EqualFold(selector, device.Username) {
		return true, true
	}
	if strings.EqualFold(selector, device.Hostname) || strings.EqualFold(selector, device.GivenName) {
		return true, true
	}
	return false, false
}

func selectorGroupMember(member, username, email string) bool {
	member = strings.TrimSpace(member)
	if strings.EqualFold(member, username) || (email != "" && strings.EqualFold(member, email)) {
		return true
	}
	if strings.HasSuffix(member, "@") {
		memberName := strings.TrimSuffix(member, "@")
		usernameName := strings.TrimSuffix(username, "@")
		if strings.EqualFold(memberName, usernameName) {
			return true
		}
		if email != "" && strings.EqualFold(memberName, strings.SplitN(email, "@", 2)[0]) {
			return true
		}
	}
	return false
}

func selectorMatchesIP(selector string, ips []string) bool {
	match, known := selectorMatchesIPValue(selector, ips)
	return known && match
}

func selectorMatchesIPValue(selector string, ips []string) (bool, bool) {
	if address, err := netip.ParseAddr(selector); err == nil {
		for _, raw := range ips {
			ip, parseErr := netip.ParseAddr(raw)
			if parseErr == nil && address == ip {
				return true, true
			}
		}
		return false, true
	}
	if prefix, err := netip.ParsePrefix(selector); err == nil {
		for _, raw := range ips {
			ip, parseErr := netip.ParseAddr(raw)
			if parseErr == nil && prefix.Contains(ip) {
				return true, true
			}
		}
		return false, true
	}
	return false, false
}

func standardJSON(raw []byte) ([]byte, error) {
	clean := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); {
		switch {
		case raw[i] == '"':
			start := i
			i++
			for i < len(raw) {
				if raw[i] == '\\' {
					i += 2
					continue
				}
				if raw[i] == '"' {
					i++
					break
				}
				i++
			}
			if i > len(raw) || raw[i-1] != '"' {
				return nil, errors.New("HuJSON contains an unterminated string")
			}
			clean = append(clean, raw[start:i]...)
		case raw[i] == '/' && i+1 < len(raw) && raw[i+1] == '/':
			i += 2
			for i < len(raw) && raw[i] != '\n' {
				i++
			}
			if i < len(raw) {
				clean = append(clean, '\n')
				i++
			}
		case raw[i] == '/' && i+1 < len(raw) && raw[i+1] == '*':
			i += 2
			closed := false
			for i+1 < len(raw) {
				if raw[i] == '*' && raw[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				if raw[i] == '\n' {
					clean = append(clean, '\n')
				}
				i++
			}
			if !closed {
				return nil, errors.New("HuJSON contains an unterminated block comment")
			}
		case raw[i] == ',':
			j := skipTrivia(raw, i+1)
			if j < len(raw) && (raw[j] == '}' || raw[j] == ']') {
				i++
				continue
			}
			clean = append(clean, raw[i])
			i++
		default:
			clean = append(clean, raw[i])
			i++
		}
	}
	return clean, nil
}

func skipTrivia(raw []byte, index int) int {
	for index < len(raw) {
		if raw[index] == ' ' || raw[index] == '\t' || raw[index] == '\r' || raw[index] == '\n' {
			index++
			continue
		}
		if index+1 < len(raw) && raw[index] == '/' && raw[index+1] == '/' {
			index += 2
			for index < len(raw) && raw[index] != '\n' {
				index++
			}
			continue
		}
		if index+1 < len(raw) && raw[index] == '/' && raw[index+1] == '*' {
			index += 2
			for index+1 < len(raw) && !(raw[index] == '*' && raw[index+1] == '/') {
				index++
			}
			if index+1 < len(raw) {
				index += 2
			}
			continue
		}
		break
	}
	return index
}

func rejectDuplicateKeys(clean []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(clean))
	decoder.UseNumber()
	var readValue func() error
	readValue = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok || seen[key] {
					return errors.New("Headscale Policy contains duplicate or invalid object keys")
				}
				seen[key] = true
				if err := readValue(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := readValue(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return errors.New("Headscale Policy contains an unexpected delimiter")
		}
	}
	if err := readValue(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err == nil {
		return errors.New("Headscale Policy contains trailing values")
	}
	return nil
}

func findRootFieldValue(raw []byte, target string) (int, int, bool, error) {
	i := skipTrivia(raw, 0)
	if i >= len(raw) || raw[i] != '{' {
		return 0, 0, false, errors.New("HuJSON policy root must be an object")
	}
	i++
	for {
		i = skipTrivia(raw, i)
		if i >= len(raw) {
			return 0, 0, false, errors.New("HuJSON object is not closed")
		}
		if raw[i] == '}' {
			return 0, 0, false, nil
		}
		keyStart, keyEnd, err := scanString(raw, i)
		if err != nil {
			return 0, 0, false, err
		}
		var key string
		if err := json.Unmarshal(raw[keyStart:keyEnd], &key); err != nil {
			return 0, 0, false, err
		}
		i = skipTrivia(raw, keyEnd)
		if i >= len(raw) || raw[i] != ':' {
			return 0, 0, false, errors.New("HuJSON object member is missing a colon")
		}
		i = skipTrivia(raw, i+1)
		valueStart := i
		valueEnd, err := scanValue(raw, valueStart)
		if err != nil {
			return 0, 0, false, err
		}
		if key == target {
			return valueStart, valueEnd, true, nil
		}
		i = skipTrivia(raw, valueEnd)
		if i < len(raw) && raw[i] == ',' {
			i++
			continue
		}
		if i < len(raw) && raw[i] == '}' {
			return 0, 0, false, nil
		}
		return 0, 0, false, errors.New("HuJSON object member separator is invalid")
	}
}

func scanString(raw []byte, start int) (int, int, error) {
	if start >= len(raw) || raw[start] != '"' {
		return 0, 0, errors.New("HuJSON object member name must be a string")
	}
	for i := start + 1; i < len(raw); i++ {
		if raw[i] == '\\' {
			i++
			continue
		}
		if raw[i] == '"' {
			return start, i + 1, nil
		}
	}
	return 0, 0, errors.New("HuJSON contains an unterminated string")
}

func scanValue(raw []byte, start int) (int, error) {
	if start >= len(raw) {
		return 0, errors.New("HuJSON value is missing")
	}
	switch raw[start] {
	case '"':
		_, end, err := scanString(raw, start)
		return end, err
	case '{', '[':
		open := raw[start]
		close := byte('}')
		if open == '[' {
			close = ']'
		}
		for i := start + 1; i < len(raw); {
			i = skipTrivia(raw, i)
			if i >= len(raw) {
				return 0, errors.New("HuJSON container is not closed")
			}
			if raw[i] == close {
				return i + 1, nil
			}
			if raw[i] == '"' {
				_, end, err := scanString(raw, i)
				if err != nil {
					return 0, err
				}
				if open == '[' {
					i = end
					continue
				}
				i = skipTrivia(raw, end)
				if i >= len(raw) || raw[i] != ':' {
					return 0, errors.New("HuJSON object member is missing a colon")
				}
				i = skipTrivia(raw, i+1)
				end, err = scanValue(raw, i)
				if err != nil {
					return 0, err
				}
				i = end
				continue
			}
			if raw[i] == '{' || raw[i] == '[' {
				end, err := scanValue(raw, i)
				if err != nil {
					return 0, err
				}
				i = end
				continue
			}
			if raw[i] == ',' {
				i++
				continue
			}
			for i < len(raw) && raw[i] != ',' && raw[i] != close && raw[i] != '}' && raw[i] != ']' {
				if raw[i] == '/' && i+1 < len(raw) && (raw[i+1] == '/' || raw[i+1] == '*') {
					break
				}
				i++
			}
		}
		return 0, errors.New("HuJSON container is not closed")
	default:
		end := start
		for end < len(raw) && raw[end] != ',' && raw[end] != '}' && raw[end] != ']' && raw[end] != ' ' && raw[end] != '\t' && raw[end] != '\r' && raw[end] != '\n' {
			end++
		}
		if end == start {
			return 0, errors.New("HuJSON value is malformed")
		}
		return end, nil
	}
}
