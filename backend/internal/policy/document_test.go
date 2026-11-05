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

func TestGroupSelectorsContinueAfterUnresolvedNestedMember(t *testing.T) {
	doc, err := Parse(`{
  "groups": {
    "group:admins": ["group:missing", "chichu@example.com"]
  },
  "grants": [{"src":["group:admins"],"dst":["100.64.0.20"],"app":{"tailscale.com/cap/relay":[]}}]
}`)
	if err != nil {
		t.Fatal(err)
	}
	state, err := doc.EffectiveRelayMatrix(
		[]DeviceSelector{{Username: "chichu", Email: "chichu@example.com", IPs: []string{"100.64.0.10"}}},
		[]DeviceSelector{{Username: "relay", IPs: []string{"100.64.0.20"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !state[0][0].Inherited || !state[0][0].Unknown {
		t.Fatalf("expected confirmed OR match with unresolved nested member: %+v", state[0][0])
	}
	if len(state[0][0].Matches) != 1 || state[0][0].Matches[0].MatchedSrc[0] != "group:admins" {
		t.Fatalf("missing selector provenance: %+v", state[0][0].Matches)
	}
}

func TestAutogroupSelfIsDestinationOnlyAndExcludesTaggedDevices(t *testing.T) {
	doc, err := Parse(`{"grants":[{"src":["autogroup:member"],"dst":["autogroup:self"],"app":{"tailscale.com/cap/relay":[]}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	sources := []DeviceSelector{{UserID: "user-1", Username: "alice", IPs: []string{"100.64.0.10"}}}
	relays := []DeviceSelector{
		{UserID: "user-1", Username: "alice-relay", IPs: []string{"100.64.0.20"}},
		{UserID: "user-1", Username: "tagged-relay", IPs: []string{"100.64.0.21"}, Tagged: true},
	}
	state, err := doc.EffectiveRelayMatrix(sources, relays)
	if err != nil {
		t.Fatal(err)
	}
	if !state[0][0].Inherited || state[0][0].Unknown {
		t.Fatalf("untagged same-user destination did not match: %+v", state[0][0])
	}
	if state[0][1].Inherited || state[0][1].Exact || state[0][1].Unknown {
		t.Fatalf("tagged destination should be excluded from autogroup:self: %+v", state[0][1])
	}

	sourceOnly, err := Parse(`{"grants":[{"src":["autogroup:self"],"dst":["100.64.0.20"],"app":{"tailscale.com/cap/relay":[]}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	state, err = sourceOnly.EffectiveRelayMatrix(sources, relays[:1])
	if err != nil {
		t.Fatal(err)
	}
	if state[0][0].Inherited || !state[0][0].Unknown {
		t.Fatalf("source-side autogroup:self must remain unresolved: %+v", state[0][0])
	}
}

func TestUserAndGroupSelectorsDoNotMatchTaggedDevices(t *testing.T) {
	doc, err := Parse(`{
  "groups": {"group:admins": ["alice@example.com"]},
  "grants": [
    {"src":["alice@example.com"],"dst":["100.64.0.20"],"app":{"tailscale.com/cap/relay":[]}},
    {"src":["group:admins"],"dst":["100.64.0.20"],"app":{"tailscale.com/cap/relay":[]}}
  ]
}`)
	if err != nil {
		t.Fatal(err)
	}
	state, err := doc.EffectiveRelayMatrix(
		[]DeviceSelector{{Username: "alice", Email: "alice@example.com", Hostname: "alice-server", Tagged: true, Tags: []string{"tag:server"}, IPs: []string{"100.64.0.10"}}},
		[]DeviceSelector{{Username: "relay", IPs: []string{"100.64.0.20"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if state[0][0].Inherited || state[0][0].Exact || state[0][0].Unknown || len(state[0][0].Matches) != 0 {
		t.Fatalf("user/group selectors matched a tagged device: %+v", state[0][0])
	}
	matched, known := doc.MatchSourceSelector("alice-server", DeviceSelector{Username: "alice", Email: "alice@example.com", Hostname: "alice-server", Tagged: true})
	if !matched || !known {
		t.Fatalf("hostname selector should still match a tagged device: matched=%v known=%v", matched, known)
	}
}

func TestEmailUserSelectorAndComplexRelayGrantStayUncertain(t *testing.T) {
	doc, err := Parse(`{"grants":[
  {"src":["alice@example.com"],"dst":["100.64.0.20"],"app":{"tailscale.com/cap/relay":[],"other/capability":[]}},
  {"src":["autogroup:danger-all"],"dst":["100.64.0.20"],"app":{"tailscale.com/cap/relay":[]}}
]}`)
	if err != nil {
		t.Fatal(err)
	}
	state, err := doc.EffectiveRelayMatrix(
		[]DeviceSelector{{Username: "alice", Email: "alice@example.com", IPs: []string{"100.64.0.10"}}},
		[]DeviceSelector{{Username: "relay", IPs: []string{"100.64.0.20"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !state[0][0].Unknown || !state[0][0].Inherited {
		t.Fatalf("expected recognized broad source and complex grant uncertainty: %+v", state[0][0])
	}
	if len(state[0][0].Matches) != 1 || state[0][0].Matches[0].Src[0] != "autogroup:danger-all" {
		t.Fatalf("complex grant was incorrectly exposed as confirmed provenance: %+v", state[0][0].Matches)
	}
	if len(state[0][0].Warnings) == 0 {
		t.Fatal("expected a warning for the broad autogroup scope")
	}
}

func TestAutogroupInternetIsExplainedAsUnresolvedExitNodeScope(t *testing.T) {
	doc, err := Parse(`{"grants":[{"src":["autogroup:member"],"dst":["autogroup:internet"],"app":{"tailscale.com/cap/relay":[]}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	state, err := doc.EffectiveRelayMatrix(
		[]DeviceSelector{{Username: "alice", IPs: []string{"100.64.0.10"}}},
		[]DeviceSelector{{Username: "exit", IPs: []string{"100.64.0.20"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !state[0][0].Unknown || state[0][0].Inherited || len(state[0][0].Warnings) != 1 || !strings.Contains(state[0][0].Warnings[0], "出口节点") {
		t.Fatalf("expected an exit-node scope explanation without confirmed Relay access: %+v", state[0][0])
	}
}
