package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/config"
	"github.com/XiaomaiTX/go-react-starter/internal/headscale"
	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"github.com/XiaomaiTX/go-react-starter/internal/policy"
	"github.com/XiaomaiTX/go-react-starter/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type peerRelayHeadscaleFake struct {
	mu     sync.Mutex
	users  []headscale.User
	nodes  []headscale.Node
	policy string
}

func (f *peerRelayHeadscaleFake) Configured() bool { return true }
func (f *peerRelayHeadscaleFake) ListUsers(context.Context) ([]headscale.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]headscale.User(nil), f.users...), nil
}
func (f *peerRelayHeadscaleFake) ListNodes(context.Context) ([]headscale.Node, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]headscale.Node(nil), f.nodes...), nil
}
func (f *peerRelayHeadscaleFake) GetPolicy(context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.policy, nil
}
func (f *peerRelayHeadscaleFake) SetPolicy(_ context.Context, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.policy = value
	return nil
}
func (f *peerRelayHeadscaleFake) replacePolicy(value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.policy = value
}

func newPeerRelayTestService(t *testing.T, rawPolicy, applicantMode string) (*PeerRelayService, *peerRelayHeadscaleFake, repository.PeerRelayRepository) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "peer-relay.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDatabase.Close() })
	if err := database.AutoMigrate(
		&model.HeadscaleUser{}, &model.Device{}, &model.RelayNode{}, &model.RelayApplication{},
		&model.RelayAssignment{}, &model.PolicyRevision{}, &model.AuditLog{}, &model.SyncState{},
		&model.PeerRelaySettings{},
	); err != nil {
		t.Fatal(err)
	}
	client := &peerRelayHeadscaleFake{
		users: []headscale.User{
			{ID: "u-alice", Username: "alice", Email: "alice@example.com"},
			{ID: "u-bob", Username: "bob", Email: "bob@example.com"},
			{ID: "u-admin", Username: "admin", Email: "admin@example.com"},
			{ID: "u-reviewer", Username: "reviewer", Email: "reviewer@example.com"},
		},
		nodes: []headscale.Node{
			{ID: "node-alice", UserID: "u-alice", Username: "alice", Hostname: "alice-laptop", IPAddresses: []string{"100.64.0.10"}},
			{ID: "node-bob", UserID: "u-bob", Username: "bob", Hostname: "bob-laptop", IPAddresses: []string{"100.64.0.11"}},
			{ID: "node-reviewer", UserID: "u-reviewer", Username: "reviewer", Hostname: "reviewer-laptop", IPAddresses: []string{"100.64.0.12"}},
			{ID: "node-relay", UserID: "u-admin", Username: "admin", Hostname: "relay-node", IPAddresses: []string{"100.64.0.20"}},
			{ID: "node-relay-2", UserID: "u-admin", Username: "admin", Hostname: "relay-node-2", IPAddresses: []string{"100.64.0.21"}},
		},
		policy: rawPolicy,
	}
	repo := repository.NewPeerRelayRepository(database)
	cfg := config.Config{
		AdminGroups: []string{"group:relay-admin"}, ReviewerGroups: []string{"group:relay-reviewer"},
		ApplicantGroups: []string{"group:relay-applicant"}, ApplicantMode: applicantMode, PolicyWrites: true, SelfServiceEnabled: true,
	}
	return NewPeerRelayService(repo, client, cfg), client, repo
}

func testPolicy() string {
	return `{
  // Keep this policy comment and the non-relay grant intact.
  "groups": {
    "group:relay-admin": ["admin@example.com"],
    "group:relay-reviewer": ["reviewer@example.com"],
    "group:relay-applicant": ["alice@example.com"]
  },
  "acls": [{"action":"accept","src":["*"],"dst":["*:*"],}],
  "grants": [{"src":["*"],"dst":["*"],"ip":["*"],}],
}`
}

func TestPeerRelayRolesApplicationsAndOwnership(t *testing.T) {
	ctx := context.Background()
	svc, _, repo := newPeerRelayTestService(t, testPolicy(), "groups")
	if err := svc.Sync(ctx); err != nil {
		t.Fatal(err)
	}

	alice, err := svc.Me(ctx, "alice")
	if err != nil || !alice.CanApply || alice.IsAdmin || alice.IsReviewer {
		t.Fatalf("alice role=%+v err=%v", alice, err)
	}
	bob, err := svc.Me(ctx, "bob")
	if err != nil || bob.CanApply {
		t.Fatalf("bob role=%+v err=%v", bob, err)
	}
	admin, err := svc.Me(ctx, "admin")
	if err != nil || !admin.IsAdmin || !admin.IsReviewer {
		t.Fatalf("admin role=%+v err=%v", admin, err)
	}
	if _, err := svc.CreateApplication(ctx, "bob", ApplicationInput{NodeID: "node-bob", DisplayName: "Bob Relay"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-member application error=%v", err)
	}
	if _, err := svc.CreateApplication(ctx, "alice", ApplicationInput{NodeID: "node-bob", DisplayName: "Not Alice's device"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("other-owner application error=%v", err)
	}
	application, err := svc.CreateApplication(ctx, "alice", ApplicationInput{NodeID: "node-alice", DisplayName: "Alice Relay", Reason: "Home uplink"})
	if err != nil || application.Status != "pending" || application.ApplicantUserID != "u-alice" {
		t.Fatalf("application=%+v err=%v", application, err)
	}
	if _, err := svc.CreateApplication(ctx, "alice", ApplicationInput{NodeID: "node-alice", DisplayName: "Duplicate"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate pending application error=%v", err)
	}
	if _, err := svc.ReviewApplication(ctx, "alice", application.ID, "approved", "", nil); !errors.Is(err, ErrForbidden) {
		t.Fatalf("applicant review error=%v", err)
	}
	approvedMetadata := &RelayInput{DisplayName: "Alice Relay Approved", Region: "Shenzhen", PublicEndpoint: "relay.example.test"}
	reviewed, err := svc.ReviewApplication(ctx, "reviewer", application.ID, "approved", "Network checked", approvedMetadata)
	if err != nil || reviewed.Status != "approved" || reviewed.Reviewer != "reviewer" {
		t.Fatalf("review=%+v err=%v", reviewed, err)
	}
	approvedRelay, err := repo.Relay(ctx, "node-alice")
	if err != nil {
		t.Fatalf("approved application did not register relay: %v", err)
	}
	if approvedRelay.DisplayName != "Alice Relay Approved" || approvedRelay.Region != "Shenzhen" || approvedRelay.PublicEndpoint != "relay.example.test" {
		t.Fatalf("approved metadata override was not applied: %+v", approvedRelay)
	}
	if err := svc.CancelApplication(ctx, "alice", application.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("completed application cancellation error=%v", err)
	}
}

func TestPeerRelayGrantCommitPreservesPolicyAndRejectsStaleHash(t *testing.T) {
	ctx := context.Background()
	svc, client, repo := newPeerRelayTestService(t, testPolicy(), "all")
	if err := svc.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RegisterRelay(ctx, "admin", "node-relay", RelayInput{DisplayName: "Primary Relay"}); err != nil {
		t.Fatal(err)
	}
	doc, err := policy.Parse(testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.UpdateAssignments(ctx, "alice", AssignmentInput{SourceNodeID: "node-alice", RelayNodeIDs: []string{"node-relay"}, BasePolicyHash: doc.Hash()})
	if err != nil || result.Added != 1 || !result.PolicyChanged {
		t.Fatalf("assignment result=%+v err=%v", result, err)
	}
	updatedPolicy, err := client.GetPolicy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updatedPolicy, "// Keep this policy comment") || !strings.Contains(updatedPolicy, `"acls": [{"action":"accept","src":["*"],"dst":["*:*"],}],`) {
		t.Fatalf("unrelated policy content was not preserved: %s", updatedPolicy)
	}
	updatedDoc, err := policy.Parse(updatedPolicy)
	if err != nil {
		t.Fatal(err)
	}
	if count, err := updatedDoc.ExactRelayGrantCount("100.64.0.10", "100.64.0.20"); err != nil || count != 1 {
		t.Fatalf("exact relay grants count=%d err=%v", count, err)
	}
	assignments, err := repo.Assignments(ctx, "node-alice")
	if err != nil || len(assignments) != 1 || !assignments[0].Managed {
		t.Fatalf("managed assignments=%+v err=%v", assignments, err)
	}
	if _, err := svc.UpdateAssignments(ctx, "alice", AssignmentInput{SourceNodeID: "node-alice", RelayNodeIDs: nil, BasePolicyHash: doc.Hash()}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale policy hash error=%v", err)
	}
	if _, err := svc.UpdateAssignments(ctx, "bob", AssignmentInput{SourceNodeID: "node-alice", RelayNodeIDs: nil, BasePolicyHash: updatedDoc.Hash()}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("cross-owner graph update error=%v", err)
	}

	// A real external change must be detected before the service writes.
	externalPolicy := strings.Replace(updatedPolicy, `"acls":`, `"hosts":{"manual":"100.64.0.20"},"acls":`, 1)
	client.replacePolicy(externalPolicy)
	if _, err := svc.UpdateAssignments(ctx, "alice", AssignmentInput{SourceNodeID: "node-alice", RelayNodeIDs: []string{"node-relay"}, BasePolicyHash: updatedDoc.Hash()}); !errors.Is(err, ErrConflict) {
		t.Fatalf("external policy change error=%v", err)
	}
}

func TestPeerRelayConcurrentPolicyWritesDetectStaleEditor(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newPeerRelayTestService(t, testPolicy(), "all")
	if err := svc.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	for _, nodeID := range []string{"node-relay", "node-relay-2"} {
		if _, err := svc.RegisterRelay(ctx, "admin", nodeID, RelayInput{DisplayName: nodeID}); err != nil {
			t.Fatal(err)
		}
	}
	doc, err := policy.Parse(testPolicy())
	if err != nil {
		t.Fatal(err)
	}
	input := AssignmentInput{SourceNodeID: "node-alice", RelayNodeIDs: []string{"node-relay"}, BasePolicyHash: doc.Hash()}
	results := make(chan error, 2)
	var wait sync.WaitGroup
	for index := 0; index < 2; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, writeErr := svc.UpdateAssignments(ctx, "alice", input)
			results <- writeErr
		}()
	}
	wait.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent write error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
}

func TestAdminSettingsCanDelegateReviewerCapabilities(t *testing.T) {
	ctx := context.Background()
	svc, _, _ := newPeerRelayTestService(t, testPolicy(), "all")
	if err := svc.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	settings, err := svc.UpdateSettings(ctx, "admin", SettingsInput{
		AdminGroups: []string{"group:relay-admin"}, ReviewerGroups: []string{"group:relay-reviewer"},
		ApplicantGroups: []string{"group:relay-applicant"}, ApplicantMode: "groups",
		ReviewerManageRelays: true, ReviewerEditAllDevices: true, SelfServiceEnabled: true, SyncIntervalSeconds: 90,
	})
	if err != nil || settings.ApplicantMode != "groups" || !settings.ReviewerManageRelays || !settings.ReviewerEditAllDevices || settings.SyncIntervalSeconds != 90 {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
	if interval := svc.syncInterval(ctx); interval != 90*time.Second {
		t.Fatalf("sync interval after settings update=%s", interval)
	}
	select {
	case <-svc.syncScheduleChanged:
	default:
		t.Fatal("sync scheduler was not notified about the new interval")
	}
	if _, err := svc.UpdateSettings(ctx, "alice", SettingsInput{
		AdminGroups: []string{"group:relay-admin"}, ApplicantMode: "all",
	}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-admin settings update error=%v", err)
	}
	if _, err := svc.UpdateSettings(ctx, "admin", SettingsInput{
		AdminGroups: []string{"group:missing"}, ReviewerGroups: []string{}, ApplicantGroups: []string{}, ApplicantMode: "all",
	}); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown group settings error=%v", err)
	}

	reviewer, err := svc.Me(ctx, "reviewer")
	if err != nil || !reviewer.IsReviewer || !reviewer.CanManageRelays || !reviewer.CanEditAllDevices || reviewer.CanApply {
		t.Fatalf("reviewer role=%+v err=%v", reviewer, err)
	}
	if _, err := svc.RegisterRelay(ctx, "reviewer", "node-relay", RelayInput{DisplayName: "Delegated Relay"}); err != nil {
		t.Fatalf("delegated relay registration failed: %v", err)
	}
	graph, err := svc.Graph(ctx, "reviewer", "node-bob")
	if err != nil || !graph.CanEdit {
		t.Fatalf("reviewer graph=%+v err=%v", graph, err)
	}
	result, err := svc.UpdateAssignments(ctx, "reviewer", AssignmentInput{
		SourceNodeID: "node-bob", RelayNodeIDs: []string{"node-relay"}, BasePolicyHash: graph.BasePolicyHash,
	})
	if err != nil || result.Added != 1 {
		t.Fatalf("delegated assignment=%+v err=%v", result, err)
	}
	if _, err := svc.UpdateSettings(ctx, "admin", SettingsInput{
		AdminGroups: []string{"group:relay-admin"}, ReviewerGroups: []string{"group:relay-reviewer"},
		ApplicantGroups: []string{"group:relay-applicant"}, ApplicantMode: "groups", ReviewerManageRelays: true, SelfServiceEnabled: false, SyncIntervalSeconds: 120,
	}); err != nil {
		t.Fatalf("disable reviewer all-device delegation: %v", err)
	}
	if _, err := svc.RegisterRelay(ctx, "reviewer", "node-bob", RelayInput{DisplayName: "Not owned"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("delegated manager registered another user's device: %v", err)
	}
	if _, err := svc.RegisterRelay(ctx, "reviewer", "node-reviewer", RelayInput{DisplayName: "Own Relay"}); err != nil {
		t.Fatalf("delegated manager could not register own device: %v", err)
	}
	graph, err = svc.Graph(ctx, "alice", "node-alice")
	if err != nil || graph.CanEdit {
		t.Fatalf("self-service disabled graph=%+v err=%v", graph, err)
	}
	if _, err := svc.UpdateAssignments(ctx, "alice", AssignmentInput{SourceNodeID: "node-alice", RelayNodeIDs: []string{}, BasePolicyHash: graph.BasePolicyHash}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("self-service disabled assignment error=%v", err)
	}
}

func TestRollbackHonorsDisabledPolicyWrites(t *testing.T) {
	ctx := context.Background()
	svc, _, repo := newPeerRelayTestService(t, testPolicy(), "all")
	if err := svc.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RegisterRelay(ctx, "admin", "node-relay", RelayInput{DisplayName: "Primary Relay"}); err != nil {
		t.Fatal(err)
	}
	doc, _ := policy.Parse(testPolicy())
	if _, err := svc.UpdateAssignments(ctx, "alice", AssignmentInput{SourceNodeID: "node-alice", RelayNodeIDs: []string{"node-relay"}, BasePolicyHash: doc.Hash()}); err != nil {
		t.Fatal(err)
	}
	revisions, err := repo.Revisions(ctx, 30)
	if err != nil || len(revisions) != 1 {
		t.Fatalf("revisions=%d err=%v", len(revisions), err)
	}
	svc.cfg.PolicyWrites = false
	before, _ := svc.headscale.GetPolicy(ctx)
	if _, err := svc.Rollback(ctx, "admin", revisions[0].ID); !errors.Is(err, ErrPolicyWrites) {
		t.Fatalf("rollback error=%v", err)
	}
	after, _ := svc.headscale.GetPolicy(ctx)
	if before != after {
		t.Fatal("policy changed while policy writes were disabled")
	}
}
