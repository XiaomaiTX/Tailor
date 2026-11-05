package policy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseHuJSONAndReplaceOnlyGrants(t *testing.T) {
	raw := `{
  // platform note must survive
  "groups": {"group:relay": ["alice@example.com",],},
  "acls": [{"action": "accept", "src": ["*"], "dst": ["*:*"],},],
  "grants": [],
}`
	doc, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	grant, err := NewRelayGrant("100.64.0.10", "100.64.0.20")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := doc.ReplaceGrants([]json.RawMessage{grant})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated, "// platform note must survive") {
		t.Fatal("comment outside grants was not preserved")
	}
	if !strings.Contains(updated, `"acls": [{"action": "accept", "src": ["*"], "dst": ["*:*"],},],`) {
		t.Fatal("non-grant policy text was changed")
	}
	parsed, err := Parse(updated)
	if err != nil || len(parsed.Grants) != 1 {
		t.Fatalf("updated policy did not parse: grants=%d err=%v", len(parsed.Grants), err)
	}
}

func TestReplaceGrantsRefusesMissingArray(t *testing.T) {
	doc, err := Parse(`{"acls": []}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.ReplaceGrants(nil); err == nil {
		t.Fatal("expected missing grants array to be rejected")
	}
}

func TestParseRejectsDuplicateKeys(t *testing.T) {
	if _, err := Parse(`{"grants": [], "grants": []}`); err == nil {
		t.Fatal("expected duplicate key rejection")
	}
}

func TestEffectiveRelayClassifiesExactInheritedAndUnknown(t *testing.T) {
	raw := `{
  "groups": {"group:relay-users": ["alice@example.com"]},
  "hosts": {"relay-host": "100.64.0.20"},
  "grants": [
    {"src": ["100.64.0.10"], "dst": ["100.64.0.20"], "app": {"tailscale.com/cap/relay": []}},
    {"src": ["group:relay-users"], "dst": ["relay-host"], "app": {"tailscale.com/cap/relay": []}},
    {"src": ["autogroup:unknown"], "dst": ["relay-host"], "app": {"tailscale.com/cap/relay": []}}
  ]
}`
	doc, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	source := DeviceSelector{Username: "alice", Email: "alice@example.com", IPs: []string{"100.64.0.10"}}
	relay := DeviceSelector{Username: "relay", IPs: []string{"100.64.0.20"}}
	exact, inherited, unknown, err := doc.EffectiveRelay(source, relay)
	if err != nil {
		t.Fatal(err)
	}
	if !exact || !inherited || !unknown {
		t.Fatalf("classification got exact=%v inherited=%v unknown=%v", exact, inherited, unknown)
	}
}

func TestExactRelayGrantCountDoesNotMatchBroaderGrant(t *testing.T) {
	doc, err := Parse(`{"grants":[{"src":["*"],"dst":["100.64.0.20"],"app":{"tailscale.com/cap/relay":[]}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	count, err := doc.ExactRelayGrantCount("100.64.0.10", "100.64.0.20")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("broader grant was counted as exact: %d", count)
	}
}
