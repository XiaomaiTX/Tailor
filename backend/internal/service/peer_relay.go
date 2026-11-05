package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/config"
	"github.com/XiaomaiTX/go-react-starter/internal/headscale"
	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"github.com/XiaomaiTX/go-react-starter/internal/policy"
	"github.com/XiaomaiTX/go-react-starter/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrForbidden    = errors.New("没有执行此操作的权限")
	ErrConflict     = errors.New("数据已变化或存在冲突")
	ErrUnavailable  = errors.New("Headscale 暂时不可用")
	ErrPolicyWrites = errors.New("Headscale Policy 写入未启用")
)

type PeerRelayService struct {
	repo                repository.PeerRelayRepository
	headscale           headscale.Client
	cfg                 config.Config
	syncMu              sync.Mutex
	policyMu            sync.Mutex
	appMu               sync.Mutex
	syncScheduleChanged chan struct{}
	now                 func() time.Time
}

func NewPeerRelayService(repo repository.PeerRelayRepository, client headscale.Client, cfg config.Config) *PeerRelayService {
	return &PeerRelayService{repo: repo, headscale: client, cfg: cfg, syncScheduleChanged: make(chan struct{}, 1), now: time.Now}
}

type UserView struct {
	Username          string   `json:"username"`
	Email             string   `json:"email,omitempty"`
	Groups            []string `json:"groups"`
	IsAdmin           bool     `json:"isAdmin"`
	IsReviewer        bool     `json:"isReviewer"`
	CanApply          bool     `json:"canApply"`
	CanManageRelays   bool     `json:"canManageRelays"`
	CanEditAllDevices bool     `json:"canEditAllDevices"`
}

type SettingsInput struct {
	AdminGroups            []string `json:"adminGroups"`
	ReviewerGroups         []string `json:"reviewerGroups"`
	ApplicantGroups        []string `json:"applicantGroups"`
	ApplicantMode          string   `json:"applicantMode"`
	ReviewerManageRelays   bool     `json:"reviewerManageRelays"`
	ReviewerEditAllDevices bool     `json:"reviewerEditAllDevices"`
	SelfServiceEnabled     bool     `json:"selfServiceEnabled"`
	SyncIntervalSeconds    int      `json:"syncIntervalSeconds"`
	ReapplyCooldownHours   int      `json:"reapplyCooldownHours"`
}

type SettingsView struct {
	AdminGroups            []string   `json:"adminGroups"`
	ReviewerGroups         []string   `json:"reviewerGroups"`
	ApplicantGroups        []string   `json:"applicantGroups"`
	ApplicantMode          string     `json:"applicantMode"`
	ReviewerManageRelays   bool       `json:"reviewerManageRelays"`
	ReviewerEditAllDevices bool       `json:"reviewerEditAllDevices"`
	SelfServiceEnabled     bool       `json:"selfServiceEnabled"`
	HeadscaleConfigured    bool       `json:"headscaleConfigured"`
	PolicyWritesEnabled    bool       `json:"policyWritesEnabled"`
	SyncIntervalSeconds    int        `json:"syncIntervalSeconds"`
	ReapplyCooldownHours   int        `json:"reapplyCooldownHours"`
	UpdatedBy              string     `json:"updatedBy,omitempty"`
	UpdatedAt              *time.Time `json:"updatedAt,omitempty"`
}

type DeviceView struct {
	NodeID    string     `json:"nodeId"`
	UserID    string     `json:"-"`
	Hostname  string     `json:"hostname"`
	GivenName string     `json:"givenName"`
	IPv4      string     `json:"ipv4,omitempty"`
	IPv6      string     `json:"ipv6,omitempty"`
	Username  string     `json:"username"`
	UserEmail string     `json:"-"`
	Tags      []string   `json:"tags"`
	Online    *bool      `json:"online,omitempty"`
	LastSeen  *time.Time `json:"lastSeen,omitempty"`
	Expiry    *time.Time `json:"expiry,omitempty"`
	Missing   bool       `json:"missing"`
}

type RelayView struct {
	model.RelayNode
	Device DeviceView `json:"device"`
}

type ApplicationInput struct {
	NodeID         string `json:"nodeId"`
	DisplayName    string `json:"displayName"`
	Region         string `json:"region,omitempty"`
	UDPPort        *int   `json:"udpPort,omitempty"`
	PublicEndpoint string `json:"publicEndpoint,omitempty"`
	ISP            string `json:"isp,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

type ApplicationView struct {
	model.RelayApplication
	Device DeviceView `json:"device"`
}

type RelayInput struct {
	DisplayName    string `json:"displayName"`
	Region         string `json:"region,omitempty"`
	UDPPort        *int   `json:"udpPort,omitempty"`
	PublicEndpoint string `json:"publicEndpoint,omitempty"`
	ISP            string `json:"isp,omitempty"`
	Description    string `json:"description,omitempty"`
	Selectable     *bool  `json:"selectable,omitempty"`
}

type AssignmentView struct {
	RelayNodeID  string              `json:"relayNodeId"`
	Relay        RelayView           `json:"relay"`
	Source       string              `json:"source"`
	Editable     bool                `json:"editable"`
	Warning      string              `json:"warning,omitempty"`
	GrantMatches []policy.GrantMatch `json:"grantMatches,omitempty"`
}

type GraphView struct {
	Device         DeviceView       `json:"device"`
	Relays         []AssignmentView `json:"relays"`
	BasePolicyHash string           `json:"basePolicyHash"`
	PolicyStale    bool             `json:"policyStale"`
	PolicyWrites   bool             `json:"policyWritesEnabled"`
	CanEdit        bool             `json:"canEdit"`
}

type CanvasNodeView struct {
	Device     DeviceView       `json:"device"`
	Relay      *model.RelayNode `json:"relay,omitempty"`
	CanEdit    bool             `json:"canEdit"`
	CanConfirm bool             `json:"canConfirm"`
}

type CanvasEdgeView struct {
	SourceNodeID string              `json:"sourceNodeId"`
	RelayNodeID  string              `json:"relayNodeId"`
	Source       string              `json:"source"`
	Editable     bool                `json:"editable"`
	Warning      string              `json:"warning,omitempty"`
	GrantMatches []policy.GrantMatch `json:"grantMatches,omitempty"`
}

type CanvasGraphView struct {
	Nodes          []CanvasNodeView `json:"nodes"`
	Edges          []CanvasEdgeView `json:"edges"`
	BasePolicyHash string           `json:"basePolicyHash"`
	PolicyStale    bool             `json:"policyStale"`
	PolicyWrites   bool             `json:"policyWritesEnabled"`
}

type DashboardView struct {
	DeviceCount     int        `json:"deviceCount"`
	RelayCount      int        `json:"relayCount"`
	SelectableCount int        `json:"selectableRelayCount"`
	AssignmentCount int        `json:"assignmentCount"`
	PendingCount    int        `json:"pendingApplicationCount"`
	LastSyncedAt    *time.Time `json:"lastSyncedAt,omitempty"`
	SyncError       string     `json:"syncError,omitempty"`
	PolicyHash      string     `json:"policyHash,omitempty"`
	PolicyWrites    bool       `json:"policyWritesEnabled"`
	CurrentUser     UserView   `json:"currentUser"`
}

type AssignmentInput struct {
	SourceNodeID   string   `json:"sourceNodeId"`
	RelayNodeIDs   []string `json:"relayNodeIds"`
	BasePolicyHash string   `json:"basePolicyHash"`
}

type AssignmentResult struct {
	Graph         GraphView `json:"graph"`
	Added         int       `json:"added"`
	Removed       int       `json:"removed"`
	PolicyChanged bool      `json:"policyChanged"`
}

type AssignmentChange struct {
	SourceNodeID string   `json:"sourceNodeId"`
	RelayNodeIDs []string `json:"relayNodeIds"`
}

type BatchAssignmentInput struct {
	Changes        []AssignmentChange `json:"changes"`
	BasePolicyHash string             `json:"basePolicyHash"`
}

type BatchAssignmentResult struct {
	BasePolicyHash     string   `json:"basePolicyHash"`
	ChangedSourceCount int      `json:"changedSourceCount"`
	Added              int      `json:"added"`
	Removed            int      `json:"removed"`
	PolicyChanged      bool     `json:"policyChanged"`
	SourceNodeIDs      []string `json:"sourceNodeIds"`
}

type SelectorRulePreviewInput struct {
	SourceSelector string `json:"sourceSelector"`
	RelayNodeID    string `json:"relayNodeId"`
}

type SelectorRuleInput struct {
	SourceSelector string `json:"sourceSelector"`
	RelayNodeID    string `json:"relayNodeId"`
	BasePolicyHash string `json:"basePolicyHash"`
	ScopeHash      string `json:"scopeHash"`
}

type SelectorRulePreview struct {
	SourceSelector string       `json:"sourceSelector"`
	RelayNodeID    string       `json:"relayNodeId"`
	RelaySelector  string       `json:"relaySelector"`
	Relay          RelayView    `json:"relay"`
	SourceDevices  []DeviceView `json:"sourceDevices"`
	BasePolicyHash string       `json:"basePolicyHash"`
	ScopeHash      string       `json:"scopeHash"`
	Dynamic        bool         `json:"dynamic"`
	Warning        string       `json:"warning,omitempty"`
}

type SelectorRuleView struct {
	ID             string       `json:"id"`
	SourceSelector string       `json:"sourceSelector"`
	RelayNodeID    string       `json:"relayNodeId"`
	RelaySelector  string       `json:"relaySelector"`
	Relay          *RelayView   `json:"relay,omitempty"`
	SourceDevices  []DeviceView `json:"sourceDevices"`
	InPolicy       bool         `json:"inPolicy"`
	TargetChanged  bool         `json:"targetChanged"`
	ReadOnly       bool         `json:"readOnly"`
	CanDelete      bool         `json:"canDelete"`
	Warning        string       `json:"warning,omitempty"`
	CreatedBy      string       `json:"createdBy"`
	CreatedAt      time.Time    `json:"createdAt"`
}

type SelectorRuleMutationResult struct {
	ID             string `json:"id"`
	SourceSelector string `json:"sourceSelector"`
	RelayNodeID    string `json:"relayNodeId"`
	RelaySelector  string `json:"relaySelector"`
	BasePolicyHash string `json:"basePolicyHash"`
}

type RevisionView struct {
	ID         string    `json:"id"`
	Actor      string    `json:"actor"`
	Summary    string    `json:"summary"`
	BeforeHash string    `json:"beforeHash"`
	AfterHash  string    `json:"afterHash"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}

type RevisionDetailView struct {
	RevisionView
	Before             string             `json:"before"`
	After              string             `json:"after"`
	AddedRelayGrants   []RelayGrantChange `json:"addedRelayGrants"`
	RemovedRelayGrants []RelayGrantChange `json:"removedRelayGrants"`
	OtherPolicyChanged bool               `json:"otherPolicyChanged"`
}

type RelayGrantChange struct {
	Src []string `json:"src"`
	Dst []string `json:"dst"`
}

type AuditView struct {
	ID        int64     `json:"id"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Before    string    `json:"before,omitempty"`
	After     string    `json:"after,omitempty"`
	Result    string    `json:"result"`
	CreatedAt time.Time `json:"createdAt"`
}

type roleContext struct {
	user               model.HeadscaleUser
	isAdmin            bool
	reviewer           bool
	canApply           bool
	canManageRelays    bool
	canEditAllDevices  bool
	selfServiceEnabled bool
	groups             []string
}

func (s *PeerRelayService) Sync(ctx context.Context) error {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	attempt := s.now().UTC()
	state, _ := s.repo.SyncState(ctx)
	state.Key = "headscale"
	state.LastAttempt = &attempt
	if !s.headscale.Configured() {
		state.LastError = "HEADSCALE_URL 和 HEADSCALE_API_KEY 尚未配置"
		_ = s.repo.SaveSyncState(ctx, state)
		return ErrUnavailable
	}
	users, err := s.headscale.ListUsers(ctx)
	if err != nil {
		return s.failSync(ctx, state, err)
	}
	nodes, err := s.headscale.ListNodes(ctx)
	if err != nil {
		return s.failSync(ctx, state, err)
	}
	rawPolicy, err := s.headscale.GetPolicy(ctx)
	if err != nil {
		return s.failSync(ctx, state, err)
	}
	doc, err := policy.Parse(rawPolicy)
	if err != nil {
		return s.failSync(ctx, state, err)
	}
	syncedAt := s.now().UTC()
	usersToStore := make([]model.HeadscaleUser, 0, len(users))
	userEmailsByID := make(map[string]string, len(users))
	for _, user := range users {
		usersToStore = append(usersToStore, model.HeadscaleUser{ID: user.ID, Username: user.Username, Email: user.Email, SyncedAt: syncedAt})
		userEmailsByID[user.ID] = user.Email
	}
	nodesToStore := make([]model.Device, 0, len(nodes))
	for _, node := range nodes {
		ipv4, ipv6 := tailnetIPs(node.IPAddresses)
		tags, _ := json.Marshal(uniqueStrings(node.Tags))
		nodesToStore = append(nodesToStore, model.Device{
			NodeID: node.ID, UserID: node.UserID, Username: node.Username, UserEmail: userEmailsByID[node.UserID], Hostname: node.Hostname,
			GivenName: node.GivenName, IPv4: ipv4, IPv6: ipv6, TagsJSON: string(tags), Online: node.Online,
			LastSeen: headscale.ParseTime(node.LastSeen), Expiry: headscale.ParseTime(node.Expiry), SyncedAt: syncedAt,
		})
	}
	if err := s.repo.SyncSnapshot(ctx, usersToStore, nodesToStore); err != nil {
		return s.failSync(ctx, state, err)
	}
	knownNodes := indexDevices(nodesToStore)
	pending, err := s.repo.Applications(ctx, "pending", "", "")
	if err != nil {
		return s.failSync(ctx, state, err)
	}
	for _, application := range pending {
		device, exists := knownNodes[application.NodeID]
		ownerMatches := exists && device.UserID == application.ApplicantUserID
		if exists && application.ApplicantUserID == "" {
			ownerMatches = device.Username == application.Applicant
		}
		if ownerMatches {
			continue
		}
		before := jsonString(application)
		application.Status = "invalidated"
		application.ReviewNote = "设备已删除或所有者已变更"
		application.UpdatedAt = syncedAt
		applicationAudit := s.auditRecord("system", "application.invalidated", application.ID, before, jsonString(application), "success")
		if err := s.repo.InvalidatePendingApplicationWithAudit(ctx, application, applicationAudit); err != nil {
			return s.failSync(ctx, state, err)
		}
	}
	state.LastSuccess = &syncedAt
	state.LastError = ""
	state.PolicyHash = doc.Hash()
	state.PolicyJSON = rawPolicy
	if err := s.repo.SaveSyncState(ctx, state); err != nil {
		return err
	}
	return nil
}

func (s *PeerRelayService) failSync(ctx context.Context, state model.SyncState, cause error) error {
	state.LastError = "无法同步 Headscale 数据"
	_ = s.repo.SaveSyncState(ctx, state)
	return fmt.Errorf("%w: %v", ErrUnavailable, cause)
}

func (s *PeerRelayService) RunSyncScheduler(ctx context.Context) {
	_ = s.Sync(ctx)
	timer := time.NewTimer(s.syncInterval(ctx))
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.syncScheduleChanged:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(s.syncInterval(ctx))
		case <-timer.C:
			_ = s.Sync(ctx)
			timer.Reset(s.syncInterval(ctx))
		}
	}
}

func (s *PeerRelayService) syncInterval(ctx context.Context) time.Duration {
	settings, err := s.accessSettings(ctx)
	if err == nil && settings.SyncIntervalSeconds >= 10 && settings.SyncIntervalSeconds <= 3600 {
		return time.Duration(settings.SyncIntervalSeconds) * time.Second
	}
	if s.cfg.SyncIntervalSeconds >= 10 && s.cfg.SyncIntervalSeconds <= 3600 {
		return time.Duration(s.cfg.SyncIntervalSeconds) * time.Second
	}
	return 60 * time.Second
}

func (s *PeerRelayService) SyncNow(ctx context.Context, username string) error {
	doc, _, _ := s.readPolicy(ctx)
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return err
	}
	if !role.isAdmin {
		return ErrForbidden
	}
	if err := s.repo.Audit(ctx, s.auditRecord(username, "sync.manual", "headscale", "", "", "pending")); err != nil {
		return err
	}
	if err := s.Sync(ctx); err != nil {
		_ = s.audit(ctx, username, "sync.manual", "headscale", "", "", "failed")
		return err
	}
	return s.audit(ctx, username, "sync.manual", "headscale", "", "", "success")
}

func (s *PeerRelayService) Me(ctx context.Context, username string) (UserView, error) {
	doc, _, err := s.readPolicy(ctx)
	if err != nil {
		return UserView{}, err
	}
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return UserView{}, err
	}
	return s.userView(role), nil
}

func (s *PeerRelayService) Settings(ctx context.Context, username string) (SettingsView, error) {
	doc, _, err := s.readPolicy(ctx)
	if err != nil && doc == nil {
		return SettingsView{}, err
	}
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return SettingsView{}, err
	}
	if !role.isAdmin {
		return SettingsView{}, ErrForbidden
	}
	return s.settingsView(ctx)
}

func (s *PeerRelayService) UpdateSettings(ctx context.Context, username string, input SettingsInput) (SettingsView, error) {
	doc, stale, err := s.readPolicy(ctx)
	if err != nil || stale || doc == nil {
		return SettingsView{}, fmt.Errorf("%w: 更新权限设置需要最新的 Headscale Policy", ErrUnavailable)
	}
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return SettingsView{}, err
	}
	if !role.isAdmin {
		return SettingsView{}, ErrForbidden
	}
	if input.ApplicantMode != "all" && input.ApplicantMode != "groups" && input.ApplicantMode != "disabled" {
		return SettingsView{}, ErrValidation
	}
	if input.SyncIntervalSeconds < 10 || input.SyncIntervalSeconds > 3600 {
		return SettingsView{}, ErrValidation
	}
	if input.ReapplyCooldownHours < 0 || input.ReapplyCooldownHours > 720 {
		return SettingsView{}, ErrValidation
	}
	adminGroups, err := normalizeConfiguredGroups(input.AdminGroups, doc.Groups, true)
	if err != nil {
		return SettingsView{}, err
	}
	reviewerGroups, err := normalizeConfiguredGroups(input.ReviewerGroups, doc.Groups, false)
	if err != nil {
		return SettingsView{}, err
	}
	applicantGroups, err := normalizeConfiguredGroups(input.ApplicantGroups, doc.Groups, false)
	if err != nil {
		return SettingsView{}, err
	}
	bootstrapAdmin := stringContainsFold(s.cfg.BootstrapAdmins, username)
	if !bootstrapAdmin && !belongsToAny(role.groups, adminGroups) {
		return SettingsView{}, fmt.Errorf("%w: 管理员设置必须保留当前管理员的 Policy Group", ErrValidation)
	}
	previous, err := s.accessSettings(ctx)
	if err != nil {
		return SettingsView{}, err
	}
	updated := model.PeerRelaySettings{
		ID: 1, AdminGroupsJSON: jsonString(adminGroups), ReviewerGroupsJSON: jsonString(reviewerGroups),
		ApplicantGroupsJSON: jsonString(applicantGroups), ApplicantMode: input.ApplicantMode,
		ReviewerManageRelays: input.ReviewerManageRelays, ReviewerEditAllDevices: input.ReviewerEditAllDevices,
		SelfServiceEnabled: input.SelfServiceEnabled, SyncIntervalSeconds: input.SyncIntervalSeconds,
		ReapplyCooldownHours: input.ReapplyCooldownHours,
		UpdatedBy:            username, UpdatedAt: s.now().UTC(),
	}
	beforeAudit, err := settingsAuditSnapshot(previous)
	if err != nil {
		return SettingsView{}, err
	}
	afterAudit, err := settingsAuditSnapshot(updated)
	if err != nil {
		return SettingsView{}, err
	}
	if err := s.repo.SavePeerRelaySettingsWithAudit(ctx, updated, s.auditRecord(username, "settings.update", "peer-relay", jsonString(beforeAudit), jsonString(afterAudit), "success")); err != nil {
		return SettingsView{}, err
	}
	select {
	case s.syncScheduleChanged <- struct{}{}:
	default:
	}
	return s.settingsView(ctx)
}

func (s *PeerRelayService) settingsView(ctx context.Context) (SettingsView, error) {
	settings, err := s.accessSettings(ctx)
	if err != nil {
		return SettingsView{}, err
	}
	adminGroups, reviewerGroups, applicantGroups, err := decodeSettingsGroups(settings)
	if err != nil {
		return SettingsView{}, err
	}
	var updatedAt *time.Time
	if !settings.UpdatedAt.IsZero() {
		updatedAt = &settings.UpdatedAt
	}
	return SettingsView{
		AdminGroups: adminGroups, ReviewerGroups: reviewerGroups, ApplicantGroups: applicantGroups,
		ApplicantMode: settings.ApplicantMode, ReviewerManageRelays: settings.ReviewerManageRelays,
		ReviewerEditAllDevices: settings.ReviewerEditAllDevices, HeadscaleConfigured: s.headscale.Configured(),
		SelfServiceEnabled: settings.SelfServiceEnabled, SyncIntervalSeconds: configuredSyncInterval(settings.SyncIntervalSeconds),
		ReapplyCooldownHours: settings.ReapplyCooldownHours,
		PolicyWritesEnabled:  s.cfg.PolicyWrites,
		UpdatedBy:            settings.UpdatedBy, UpdatedAt: updatedAt,
	}, nil
}

func (s *PeerRelayService) accessSettings(ctx context.Context) (model.PeerRelaySettings, error) {
	settings, err := s.repo.PeerRelaySettings(ctx)
	if err == nil {
		return settings, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return model.PeerRelaySettings{}, err
	}
	return model.PeerRelaySettings{
		ID: 1, AdminGroupsJSON: jsonString(s.cfg.AdminGroups), ReviewerGroupsJSON: jsonString(s.cfg.ReviewerGroups),
		ApplicantGroupsJSON: jsonString(s.cfg.ApplicantGroups), ApplicantMode: s.cfg.ApplicantMode,
		SelfServiceEnabled: s.cfg.SelfServiceEnabled, SyncIntervalSeconds: configuredSyncInterval(s.cfg.SyncIntervalSeconds),
		ReapplyCooldownHours: 0,
		UpdatedBy:            "environment",
	}, nil
}

func configuredSyncInterval(seconds int) int {
	if seconds < 10 || seconds > 3600 {
		return 60
	}
	return seconds
}

func decodeSettingsGroups(settings model.PeerRelaySettings) ([]string, []string, []string, error) {
	var adminGroups, reviewerGroups, applicantGroups []string
	if err := json.Unmarshal([]byte(settings.AdminGroupsJSON), &adminGroups); err != nil {
		return nil, nil, nil, err
	}
	if err := json.Unmarshal([]byte(settings.ReviewerGroupsJSON), &reviewerGroups); err != nil {
		return nil, nil, nil, err
	}
	if err := json.Unmarshal([]byte(settings.ApplicantGroupsJSON), &applicantGroups); err != nil {
		return nil, nil, nil, err
	}
	return adminGroups, reviewerGroups, applicantGroups, nil
}

func settingsAuditSnapshot(settings model.PeerRelaySettings) (map[string]any, error) {
	adminGroups, reviewerGroups, applicantGroups, err := decodeSettingsGroups(settings)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"adminGroups": adminGroups, "reviewerGroups": reviewerGroups, "applicantGroups": applicantGroups,
		"applicantMode": settings.ApplicantMode, "reviewerManageRelays": settings.ReviewerManageRelays,
		"reviewerEditAllDevices": settings.ReviewerEditAllDevices, "selfServiceEnabled": settings.SelfServiceEnabled,
		"syncIntervalSeconds": settings.SyncIntervalSeconds, "reapplyCooldownHours": settings.ReapplyCooldownHours,
		"updatedBy": settings.UpdatedBy, "updatedAt": settings.UpdatedAt,
	}, nil
}

func normalizeConfiguredGroups(values []string, policyGroups map[string][]string, required bool) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || !strings.HasPrefix(value, "group:") || len([]rune(value)) > 120 {
			return nil, ErrValidation
		}
		if _, exists := policyGroups[value]; !exists {
			return nil, fmt.Errorf("%w: Policy 中不存在 Group %s", ErrValidation, value)
		}
		key := strings.ToLower(value)
		if !seen[key] {
			seen[key] = true
			result = append(result, value)
		}
	}
	if required && len(result) == 0 {
		return nil, ErrValidation
	}
	if len(result) > 20 {
		return nil, ErrValidation
	}
	sort.Strings(result)
	return result, nil
}

func (s *PeerRelayService) Dashboard(ctx context.Context, username string) (DashboardView, error) {
	doc, stale, _ := s.readPolicy(ctx)
	role, roleErr := s.roles(ctx, username, doc)
	if roleErr != nil {
		return DashboardView{}, roleErr
	}
	devices, err := s.repo.Devices(ctx)
	if err != nil {
		return DashboardView{}, err
	}
	relays, err := s.repo.Relays(ctx, role.canManageRelays)
	if err != nil {
		return DashboardView{}, err
	}
	applications, err := s.repo.Applications(ctx, "pending", "", "")
	if err != nil {
		return DashboardView{}, err
	}
	assignments, err := s.repo.Assignments(ctx, "")
	if err != nil {
		return DashboardView{}, err
	}
	usersCount, relaysCount, selectableCount := 0, 0, 0
	visibleDevices := make(map[string]bool, len(devices))
	for _, device := range devices {
		if role.canEditAllDevices || device.UserID == role.user.ID {
			usersCount++
			visibleDevices[device.NodeID] = true
		}
	}
	for _, relay := range relays {
		if relay.CancelledAt == nil {
			relaysCount++
			if relay.Selectable {
				selectableCount++
			}
		}
	}
	ownedPending := 0
	if role.reviewer {
		ownedPending = len(applications)
	} else {
		for _, application := range applications {
			if application.ApplicantUserID == role.user.ID || (application.ApplicantUserID == "" && application.Applicant == username) {
				ownedPending++
			}
		}
	}
	assignmentCount := 0
	for _, assignment := range assignments {
		if visibleDevices[assignment.SourceNodeID] {
			assignmentCount++
		}
	}
	state, _ := s.repo.SyncState(ctx)
	view := DashboardView{DeviceCount: usersCount, RelayCount: relaysCount, SelectableCount: selectableCount, AssignmentCount: assignmentCount, PendingCount: ownedPending, LastSyncedAt: state.LastSuccess, SyncError: state.LastError, PolicyHash: state.PolicyHash, PolicyWrites: s.cfg.PolicyWrites, CurrentUser: s.userView(role)}
	if doc != nil && !stale {
		view.PolicyHash = doc.Hash()
	}
	return view, nil
}

func (s *PeerRelayService) Devices(ctx context.Context, username, query string) ([]DeviceView, error) {
	doc, _, _ := s.readPolicy(ctx)
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return nil, err
	}
	devices, err := s.repo.Devices(ctx)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	result := make([]DeviceView, 0, len(devices))
	for _, device := range devices {
		if !role.canEditAllDevices && device.UserID != role.user.ID {
			continue
		}
		view := deviceView(device)
		text := strings.ToLower(strings.Join([]string{view.Hostname, view.GivenName, view.Username, view.IPv4, view.IPv6, strings.Join(view.Tags, " ")}, " "))
		if query != "" && !strings.Contains(text, query) {
			continue
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *PeerRelayService) Relays(ctx context.Context, username string, all bool) ([]RelayView, error) {
	doc, _, _ := s.readPolicy(ctx)
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return nil, err
	}
	if all && !role.canManageRelays {
		return nil, ErrForbidden
	}
	return s.relayViews(ctx, all)
}

func (s *PeerRelayService) relayViews(ctx context.Context, all bool) ([]RelayView, error) {
	relays, err := s.repo.Relays(ctx, all)
	if err != nil {
		return nil, err
	}
	result := make([]RelayView, 0, len(relays))
	for _, relay := range relays {
		if relay.CancelledAt != nil || (!all && !relay.Selectable) {
			continue
		}
		device, deviceErr := s.repo.Device(ctx, relay.NodeID)
		if deviceErr != nil && !errors.Is(deviceErr, repository.ErrNotFound) {
			return nil, deviceErr
		}
		result = append(result, RelayView{RelayNode: relay, Device: deviceView(device)})
	}
	return result, nil
}

func (s *PeerRelayService) RegisterRelay(ctx context.Context, username, nodeID string, input RelayInput) (RelayView, error) {
	doc, _, role, err := s.prepareWrite(ctx, username)
	if err != nil {
		return RelayView{}, err
	}
	_ = doc
	if !role.canManageRelays {
		return RelayView{}, ErrForbidden
	}
	device, err := s.repo.Device(ctx, nodeID)
	if err != nil || device.Missing {
		return RelayView{}, repository.ErrNotFound
	}
	if !role.isAdmin && !role.canEditAllDevices && device.UserID != role.user.ID {
		return RelayView{}, ErrForbidden
	}
	if _, err := s.repo.Relay(ctx, nodeID); err == nil {
		return RelayView{}, ErrConflict
	} else if !errors.Is(err, repository.ErrNotFound) {
		return RelayView{}, err
	}
	normalized, err := normalizeRelayInput(input)
	if err != nil {
		return RelayView{}, err
	}
	selectable := true
	if input.Selectable != nil {
		selectable = *input.Selectable
	}
	now := s.now().UTC()
	relay := model.RelayNode{NodeID: nodeID, DisplayName: normalized.DisplayName, Region: normalized.Region, UDPPort: normalized.UDPPort, PublicEndpoint: normalized.PublicEndpoint, ISP: normalized.ISP, Description: normalized.Description, Selectable: selectable, CreatedBy: username, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.SaveRelayWithAudit(ctx, relay, s.auditRecord(username, "relay.register", nodeID, "", jsonString(relay), "success")); err != nil {
		return RelayView{}, err
	}
	return RelayView{RelayNode: relay, Device: deviceView(device)}, nil
}

func (s *PeerRelayService) UpdateRelay(ctx context.Context, username, nodeID string, input RelayInput) (RelayView, error) {
	_, _, role, err := s.prepareWrite(ctx, username)
	if err != nil {
		return RelayView{}, err
	}
	if !role.canManageRelays {
		return RelayView{}, ErrForbidden
	}
	relay, err := s.repo.Relay(ctx, nodeID)
	if err != nil {
		return RelayView{}, err
	}
	before := jsonString(relay)
	normalized, err := normalizeRelayInput(input)
	if err != nil {
		return RelayView{}, err
	}
	relay.DisplayName, relay.Region, relay.UDPPort = normalized.DisplayName, normalized.Region, normalized.UDPPort
	relay.PublicEndpoint, relay.ISP, relay.Description = normalized.PublicEndpoint, normalized.ISP, normalized.Description
	if input.Selectable != nil {
		relay.Selectable = *input.Selectable
	}
	relay.UpdatedAt = s.now().UTC()
	if err := s.repo.SaveRelayWithAudit(ctx, relay, s.auditRecord(username, "relay.update", nodeID, before, jsonString(relay), "success")); err != nil {
		return RelayView{}, err
	}
	device, _ := s.repo.Device(ctx, nodeID)
	return RelayView{RelayNode: relay, Device: deviceView(device)}, nil
}

func (s *PeerRelayService) CancelRelay(ctx context.Context, username, nodeID string) error {
	doc, _, role, err := s.prepareWrite(ctx, username)
	if err != nil {
		return err
	}
	if !role.canManageRelays {
		return ErrForbidden
	}
	relay, err := s.repo.Relay(ctx, nodeID)
	if err != nil {
		return err
	}
	if relay.CancelledAt != nil {
		return nil
	}
	assignments, err := s.repo.Assignments(ctx, "")
	if err != nil {
		return err
	}
	links := 0
	for _, assignment := range assignments {
		if assignment.RelayNodeID == nodeID {
			links++
		}
	}
	selectorRules, err := s.repo.SelectorPolicyRules(ctx)
	if err != nil {
		return err
	}
	for _, rule := range selectorRules {
		if rule.RelayNodeID == nodeID {
			links++
		}
	}
	pendingApplications, err := s.repo.Applications(ctx, "pending", "", "")
	if err != nil {
		return err
	}
	for _, application := range pendingApplications {
		if application.NodeID == nodeID {
			links++
		}
	}
	if doc != nil {
		nodes, devicesErr := s.repo.Devices(ctx)
		if devicesErr != nil {
			return fmt.Errorf("%w: 无法检查 Relay 的现存 Policy 授权", ErrUnavailable)
		}
		deviceByID := indexDevices(nodes)
		relayDevice := deviceByID[nodeID]
		if relayDevice.NodeID == "" || relayDevice.Missing {
			return fmt.Errorf("%w: Relay 设备已缺失，无法确认外部 Policy 授权", ErrConflict)
		}
		for _, source := range deviceByID {
			exact, inherited, unknown, matchErr := doc.EffectiveRelay(selector(source), selector(relayDevice))
			if matchErr != nil {
				return fmt.Errorf("%w: 无法安全解析现有 Policy 授权", ErrConflict)
			}
			if exact || inherited || unknown {
				links++
			}
		}
	}
	if links > 0 {
		return fmt.Errorf("%w: %d 条现存或受外部规则影响的关系仍指向此 Relay，请先检查授权图", ErrConflict, links)
	}
	now := s.now().UTC()
	before := jsonString(relay)
	relay.CancelledAt = &now
	relay.Selectable = false
	relay.UpdatedAt = now
	if err := s.repo.SaveRelayWithAudit(ctx, relay, s.auditRecord(username, "relay.cancel", nodeID, before, jsonString(relay), "success")); err != nil {
		return err
	}
	return nil
}

func (s *PeerRelayService) CreateApplication(ctx context.Context, username string, input ApplicationInput) (ApplicationView, error) {
	_, _, role, err := s.prepareWrite(ctx, username)
	if err != nil {
		return ApplicationView{}, err
	}
	if !role.canApply {
		return ApplicationView{}, ErrForbidden
	}
	normalized, err := normalizeApplicationInput(input)
	if err != nil {
		return ApplicationView{}, err
	}
	s.appMu.Lock()
	defer s.appMu.Unlock()
	device, err := s.repo.Device(ctx, normalized.NodeID)
	if err != nil || device.Missing || device.UserID != role.user.ID || device.TagsJSON != "" && device.TagsJSON != "[]" {
		return ApplicationView{}, ErrForbidden
	}
	if _, err := s.repo.Relay(ctx, device.NodeID); err == nil {
		return ApplicationView{}, ErrConflict
	} else if !errors.Is(err, repository.ErrNotFound) {
		return ApplicationView{}, err
	}
	if _, err := s.repo.PendingApplication(ctx, device.NodeID); err == nil {
		return ApplicationView{}, ErrConflict
	} else if !errors.Is(err, repository.ErrNotFound) {
		return ApplicationView{}, err
	}
	now := s.now().UTC()
	application := model.RelayApplication{ID: uuid.NewString(), Applicant: username, ApplicantUserID: role.user.ID, NodeID: device.NodeID, DisplayName: normalized.DisplayName, Region: normalized.Region, UDPPort: normalized.UDPPort, PublicEndpoint: normalized.PublicEndpoint, ISP: normalized.ISP, Reason: normalized.Reason, Status: "pending", CreatedAt: now, UpdatedAt: now}
	settings, err := s.accessSettings(ctx)
	if err != nil {
		return ApplicationView{}, err
	}
	if settings.ReapplyCooldownHours > 0 {
		lastRejected, lookupErr := s.repo.LastRejectedApplication(ctx, device.NodeID)
		if lookupErr == nil && now.Before(lastRejected.UpdatedAt.Add(time.Duration(settings.ReapplyCooldownHours)*time.Hour)) {
			return ApplicationView{}, fmt.Errorf("%w: 该设备仍处于重新申请冷却期", ErrConflict)
		} else if lookupErr != nil && !errors.Is(lookupErr, repository.ErrNotFound) {
			return ApplicationView{}, lookupErr
		}
	}
	audit := s.auditRecord(username, "application.create", application.ID, "", jsonString(application), "success")
	if err := s.repo.CreateApplicationWithAudit(ctx, application, audit); err != nil {
		if _, pendingErr := s.repo.PendingApplication(ctx, device.NodeID); pendingErr == nil {
			return ApplicationView{}, ErrConflict
		}
		return ApplicationView{}, err
	}
	return ApplicationView{RelayApplication: application, Device: deviceView(device)}, nil
}

func (s *PeerRelayService) Applications(ctx context.Context, username, status string) ([]ApplicationView, error) {
	doc, _, _ := s.readPolicy(ctx)
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return nil, err
	}
	if status != "" && status != "pending" && status != "approved" && status != "rejected" && status != "cancelled" && status != "invalidated" {
		return nil, ErrValidation
	}
	applicant := ""
	applicantUserID := ""
	if !role.reviewer {
		applicant = username
		applicantUserID = role.user.ID
	}
	applications, err := s.repo.Applications(ctx, status, applicant, applicantUserID)
	if err != nil {
		return nil, err
	}
	result := make([]ApplicationView, 0, len(applications))
	for _, application := range applications {
		device, deviceErr := s.repo.Device(ctx, application.NodeID)
		if deviceErr != nil && !errors.Is(deviceErr, repository.ErrNotFound) {
			return nil, deviceErr
		}
		result = append(result, ApplicationView{RelayApplication: application, Device: deviceView(device)})
	}
	return result, nil
}

func (s *PeerRelayService) CancelApplication(ctx context.Context, username, id string) error {
	_, _, role, err := s.prepareWrite(ctx, username)
	if err != nil {
		return err
	}
	s.appMu.Lock()
	defer s.appMu.Unlock()
	application, err := s.repo.Application(ctx, id)
	if err != nil {
		return err
	}
	ownerMatches := application.ApplicantUserID != "" && application.ApplicantUserID == role.user.ID
	if application.ApplicantUserID == "" {
		ownerMatches = strings.EqualFold(application.Applicant, username)
	}
	if !ownerMatches {
		return ErrForbidden
	}
	if application.Status != "pending" {
		return ErrConflict
	}
	before := jsonString(application)
	application.Status = "cancelled"
	application.UpdatedAt = s.now().UTC()
	if err := s.repo.CancelPendingApplicationWithAudit(ctx, id, application.UpdatedAt, s.auditRecord(username, "application.cancel", id, before, jsonString(application), "success")); err != nil {
		if errors.Is(err, repository.ErrApplicationNotPending) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (s *PeerRelayService) ReviewApplication(ctx context.Context, username, id, decision, note string, override *RelayInput) (ApplicationView, error) {
	_, _, role, err := s.prepareWrite(ctx, username)
	if err != nil {
		return ApplicationView{}, err
	}
	if !role.reviewer {
		return ApplicationView{}, ErrForbidden
	}
	if decision != "approved" && decision != "rejected" {
		return ApplicationView{}, ErrValidation
	}
	if decision == "rejected" && strings.TrimSpace(note) == "" {
		return ApplicationView{}, ErrValidation
	}
	s.appMu.Lock()
	defer s.appMu.Unlock()
	application, err := s.repo.Application(ctx, id)
	if err != nil {
		return ApplicationView{}, err
	}
	if application.Status != "pending" {
		return ApplicationView{}, ErrConflict
	}
	device, err := s.repo.Device(ctx, application.NodeID)
	ownerMatches := err == nil && device.UserID == application.ApplicantUserID
	if err == nil && application.ApplicantUserID == "" {
		ownerMatches = device.Username == application.Applicant
	}
	if err != nil || device.Missing || !ownerMatches || device.TagsJSON != "" && device.TagsJSON != "[]" {
		application.Status = "invalidated"
		application.UpdatedAt = s.now().UTC()
		if invalidateErr := s.repo.InvalidatePendingApplicationWithAudit(ctx, application, s.auditRecord("system", "application.invalidated", application.ID, "", jsonString(application), "success")); invalidateErr != nil {
			return ApplicationView{}, invalidateErr
		}
		return ApplicationView{}, ErrConflict
	}
	var approvedRelay *model.RelayNode
	if decision == "approved" {
		if _, err := s.repo.Relay(ctx, application.NodeID); err == nil {
			return ApplicationView{}, ErrConflict
		} else if !errors.Is(err, repository.ErrNotFound) {
			return ApplicationView{}, err
		}
		relayInput := RelayInput{DisplayName: application.DisplayName, Region: application.Region, UDPPort: application.UDPPort, PublicEndpoint: application.PublicEndpoint, ISP: application.ISP, Description: application.Reason}
		if override != nil {
			relayInput = *override
		}
		normalized, err := normalizeRelayInput(relayInput)
		if err != nil {
			return ApplicationView{}, err
		}
		now := s.now().UTC()
		approvedRelay = &model.RelayNode{NodeID: device.NodeID, DisplayName: normalized.DisplayName, Region: normalized.Region, UDPPort: normalized.UDPPort, PublicEndpoint: normalized.PublicEndpoint, ISP: normalized.ISP, Description: normalized.Description, Selectable: true, CreatedBy: username, CreatedAt: now, UpdatedAt: now}
	}
	before := jsonString(application)
	now := s.now().UTC()
	application.Status, application.Reviewer, application.ReviewNote = decision, username, strings.TrimSpace(note)
	application.ReviewedAt, application.UpdatedAt = &now, now
	if err := s.repo.CompleteApplicationReview(ctx, application, approvedRelay, s.auditRecord(username, "application."+decision, id, before, jsonString(application), "success")); err != nil {
		if errors.Is(err, repository.ErrApplicationNotPending) {
			return ApplicationView{}, ErrConflict
		}
		return ApplicationView{}, err
	}
	return ApplicationView{RelayApplication: application, Device: deviceView(device)}, nil
}

func (s *PeerRelayService) Graph(ctx context.Context, username, sourceNodeID string) (GraphView, error) {
	doc, stale, readErr := s.readPolicy(ctx)
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return GraphView{}, err
	}
	source, err := s.repo.Device(ctx, sourceNodeID)
	if err != nil {
		return GraphView{}, err
	}
	if !role.canEditAllDevices && source.UserID != role.user.ID {
		return GraphView{}, ErrForbidden
	}
	if source.Missing {
		return GraphView{}, repository.ErrNotFound
	}
	relays, err := s.relayViews(ctx, true)
	if err != nil {
		return GraphView{}, err
	}
	assignments, err := s.repo.Assignments(ctx, sourceNodeID)
	if err != nil {
		return GraphView{}, err
	}
	byRelay := map[string]model.RelayAssignment{}
	for _, assignment := range assignments {
		byRelay[assignment.RelayNodeID] = assignment
	}
	canEdit := role.isAdmin || (len(deviceView(source).Tags) == 0 && (role.canEditAllDevices || (role.selfServiceEnabled && source.UserID == role.user.ID)))
	result := GraphView{Device: deviceView(source), Relays: []AssignmentView{}, PolicyWrites: s.cfg.PolicyWrites, CanEdit: canEdit, PolicyStale: stale || readErr != nil}
	if doc != nil {
		result.BasePolicyHash = doc.Hash()
	}
	if stale || readErr != nil {
		result.CanEdit = false
	}
	relaySelectors := make([]policy.DeviceSelector, len(relays))
	for i := range relays {
		relaySelectors[i] = selectorFromView(relays[i].Device)
	}
	var relations []policy.RelayRelationState
	if doc != nil {
		matrix, matrixErr := doc.EffectiveRelayMatrix([]policy.DeviceSelector{selector(source)}, relaySelectors)
		if matrixErr != nil {
			result.CanEdit, result.PolicyStale = false, true
		} else if len(matrix) > 0 {
			relations = matrix[0]
		}
	}
	for relayIndex, relay := range relays {
		assignment, hasAssignment := byRelay[relay.NodeID]
		kind, warning := "", ""
		var grantMatches []policy.GrantMatch
		if doc != nil && !source.Missing && !relay.Device.Missing {
			if relayIndex < len(relations) {
				relation := relations[relayIndex]
				grantMatches = relation.Matches
				if hasAssignment && assignment.Managed && relation.Exact {
					kind = "platform"
				} else if relation.Exact {
					kind = "external"
				} else if relation.Inherited {
					kind = "inherited"
				} else if relation.Unknown {
					kind = "unknown"
				}
				warning = strings.Join(relation.Warnings, "；")
				if relation.Unknown && warning == "" {
					warning = "Policy 中存在无法完整评估的 Peer Relay 选择器"
				}
				if kind == "" && hasAssignment && assignment.Managed {
					kind, warning = "exception", "平台记录存在，但当前 Policy 中未找到对应授权"
				}
			}
		}
		if kind == "" && hasAssignment && assignment.Managed {
			kind, warning = "exception", "平台记录存在，但无法确认当前 Policy 授权"
		}
		if kind == "" && !relay.Selectable && !hasAssignment {
			continue
		}
		editable := kind == "platform" || kind == "" || kind == "exception"
		if !relay.Selectable && kind == "" {
			editable = false
		}
		result.Relays = append(result.Relays, AssignmentView{RelayNodeID: relay.NodeID, Relay: relay, Source: kind, Editable: editable, Warning: warning, GrantMatches: grantMatches})
	}
	sort.Slice(result.Relays, func(i, j int) bool { return result.Relays[i].Relay.DisplayName < result.Relays[j].Relay.DisplayName })
	return result, nil
}

func (s *PeerRelayService) CanvasGraph(ctx context.Context, username string) (CanvasGraphView, error) {
	doc, stale, readErr := s.readPolicy(ctx)
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return CanvasGraphView{}, err
	}
	devices, err := s.repo.Devices(ctx)
	if err != nil {
		return CanvasGraphView{}, err
	}
	relays, err := s.repo.Relays(ctx, true)
	if err != nil {
		return CanvasGraphView{}, err
	}
	assignments, err := s.repo.Assignments(ctx, "")
	if err != nil {
		return CanvasGraphView{}, err
	}

	policyStale := stale || readErr != nil || doc == nil
	result := CanvasGraphView{
		Nodes: []CanvasNodeView{}, Edges: []CanvasEdgeView{}, PolicyWrites: s.cfg.PolicyWrites,
		PolicyStale: policyStale,
	}
	if doc != nil {
		result.BasePolicyHash = doc.Hash()
	}

	devicesByID := indexDevices(devices)
	relaysByID := make(map[string]model.RelayNode, len(relays))
	for _, relay := range relays {
		if relay.CancelledAt == nil {
			relaysByID[relay.NodeID] = relay
		}
	}
	visibleSources := make(map[string]model.Device)
	nodesByID := make(map[string]CanvasNodeView)
	for _, device := range devices {
		if role.canEditAllDevices || device.UserID == role.user.ID {
			visibleSources[device.NodeID] = device
			nodesByID[device.NodeID] = CanvasNodeView{Device: deviceView(device)}
		}
	}
	for nodeID, relay := range relaysByID {
		device, exists := devicesByID[nodeID]
		if !exists {
			device = model.Device{NodeID: nodeID, Missing: true}
		}
		node, alreadyVisible := nodesByID[nodeID]
		if !alreadyVisible {
			node.Device = deviceView(device)
		}
		relayCopy := relay
		node.Relay = &relayCopy
		nodesByID[nodeID] = node
	}

	canConfirm := func(device model.Device) bool {
		if device.Missing || policyStale || doc == nil {
			return false
		}
		_, selectorErr := selectorIP(selector(device))
		return selectorErr == nil
	}
	canEditSource := func(device model.Device) bool {
		view := deviceView(device)
		permission := role.isAdmin || role.canEditAllDevices || (role.selfServiceEnabled && device.UserID == role.user.ID)
		if len(view.Tags) > 0 && !role.isAdmin {
			permission = false
		}
		return permission && canConfirm(device) && s.cfg.PolicyWrites
	}
	for nodeID, node := range nodesByID {
		device, exists := devicesByID[nodeID]
		if !exists {
			device = model.Device{NodeID: nodeID, Missing: true}
		}
		node.CanConfirm = canConfirm(device)
		if _, isSource := visibleSources[nodeID]; isSource {
			node.CanEdit = canEditSource(device)
		}
		nodesByID[nodeID] = node
	}

	assignmentsByPair := make(map[string]model.RelayAssignment, len(assignments))
	for _, assignment := range assignments {
		assignmentsByPair[assignment.SourceNodeID+"\x00"+assignment.RelayNodeID] = assignment
	}
	sourceIDs := make([]string, 0, len(visibleSources))
	for nodeID := range visibleSources {
		sourceIDs = append(sourceIDs, nodeID)
	}
	sort.Strings(sourceIDs)
	relayIDs := make([]string, 0, len(relaysByID))
	for nodeID := range relaysByID {
		relayIDs = append(relayIDs, nodeID)
	}
	sort.Strings(relayIDs)
	sourceSelectors := make([]policy.DeviceSelector, len(sourceIDs))
	for i, sourceID := range sourceIDs {
		sourceSelectors[i] = selector(visibleSources[sourceID])
	}
	relaySelectors := make([]policy.DeviceSelector, len(relayIDs))
	for i, relayID := range relayIDs {
		relayDevice, exists := devicesByID[relayID]
		if !exists {
			relayDevice = model.Device{NodeID: relayID, Missing: true}
		}
		relaySelectors[i] = selector(relayDevice)
	}
	var relationMatrix [][]policy.RelayRelationState
	var relationErr error
	if doc != nil {
		relationMatrix, relationErr = doc.EffectiveRelayMatrix(sourceSelectors, relaySelectors)
		if relationErr != nil {
			policyStale = true
			result.PolicyStale = true
		}
	}
	for sourceIndex, sourceID := range sourceIDs {
		source := visibleSources[sourceID]
		for relayIndex, relayID := range relayIDs {
			relayDevice, exists := devicesByID[relayID]
			if !exists {
				relayDevice = model.Device{NodeID: relayID, Missing: true}
			}
			assignment, hasAssignment := assignmentsByPair[sourceID+"\x00"+relayID]
			kind, warning := "", ""
			if relationErr != nil {
				if hasAssignment && assignment.Managed {
					kind, warning = "exception", "无法解析 Policy，无法确认授权关系"
				}
			} else if doc != nil && !source.Missing && !relayDevice.Missing {
				relation := relationMatrix[sourceIndex][relayIndex]
				if hasAssignment && assignment.Managed && relation.Exact {
					kind = "platform"
				} else if relation.Exact {
					kind = "external"
				} else if relation.Inherited {
					kind = "inherited"
				} else if relation.Unknown {
					kind = "unknown"
				} else if hasAssignment && assignment.Managed {
					kind, warning = "exception", "平台记录存在，但当前 Policy 中未找到对应授权"
				}
				if relation.Unknown && warning == "" {
					warning = strings.Join(relation.Warnings, "；")
					if warning == "" {
						warning = "Policy 中存在无法完整评估的 Peer Relay 选择器"
					}
				}
			} else if hasAssignment && assignment.Managed {
				kind, warning = "exception", "设备或 Policy 数据不可用，无法确认授权关系"
			}
			if kind == "" {
				continue
			}
			editable := nodesByID[sourceID].CanEdit && (kind == "platform" || kind == "exception") && !policyStale && canConfirm(source) && canConfirm(relayDevice)
			result.Edges = append(result.Edges, CanvasEdgeView{
				SourceNodeID: sourceID, RelayNodeID: relayID, Source: kind, Editable: editable, Warning: warning,
				GrantMatches: relationMatrix[sourceIndex][relayIndex].Matches,
			})
		}
	}
	for nodeID, node := range nodesByID {
		if policyStale {
			node.CanEdit = false
			node.CanConfirm = false
		}
		nodesByID[nodeID] = node
	}
	if policyStale {
		for i := range result.Edges {
			result.Edges[i].Editable = false
		}
	}
	for _, node := range nodesByID {
		result.Nodes = append(result.Nodes, node)
	}
	sort.Slice(result.Nodes, func(i, j int) bool {
		left := result.Nodes[i].Device.GivenName
		if left == "" {
			left = result.Nodes[i].Device.Hostname
		}
		right := result.Nodes[j].Device.GivenName
		if right == "" {
			right = result.Nodes[j].Device.Hostname
		}
		if left == right {
			return result.Nodes[i].Device.NodeID < result.Nodes[j].Device.NodeID
		}
		return left < right
	})
	sort.Slice(result.Edges, func(i, j int) bool {
		if result.Edges[i].SourceNodeID == result.Edges[j].SourceNodeID {
			return result.Edges[i].RelayNodeID < result.Edges[j].RelayNodeID
		}
		return result.Edges[i].SourceNodeID < result.Edges[j].SourceNodeID
	})
	return result, nil
}

func (s *PeerRelayService) UpdateAssignments(ctx context.Context, username string, input AssignmentInput) (AssignmentResult, error) {
	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	doc, _, role, err := s.prepareWrite(ctx, username)
	if err != nil {
		return AssignmentResult{}, err
	}
	if !s.cfg.PolicyWrites {
		return AssignmentResult{}, ErrPolicyWrites
	}
	if source, getErr := s.repo.Device(ctx, input.SourceNodeID); getErr != nil || (!role.canEditAllDevices && source.UserID != role.user.ID) {
		return AssignmentResult{}, ErrForbidden
	} else if len(deviceView(source).Tags) > 0 && !role.isAdmin {
		return AssignmentResult{}, ErrForbidden
	} else if !role.isAdmin && !role.canEditAllDevices && !role.selfServiceEnabled {
		return AssignmentResult{}, ErrForbidden
	}
	if strings.TrimSpace(input.BasePolicyHash) == "" || doc.Hash() != input.BasePolicyHash {
		return AssignmentResult{}, fmt.Errorf("%w: Headscale Policy 已更新，请重新载入画布后再保存", ErrConflict)
	}
	source, err := s.repo.Device(ctx, input.SourceNodeID)
	if err != nil || source.Missing {
		return AssignmentResult{}, repository.ErrNotFound
	}
	if _, err := selectorIP(selector(source)); err != nil {
		return AssignmentResult{}, fmt.Errorf("%w: 目标设备缺少可用于 Policy 的 Tailscale IP", ErrValidation)
	}
	desired := map[string]bool{}
	for _, relayID := range input.RelayNodeIDs {
		if relayID == input.SourceNodeID {
			return AssignmentResult{}, ErrValidation
		}
		desired[relayID] = true
	}
	oldRows, err := s.repo.Assignments(ctx, input.SourceNodeID)
	if err != nil {
		return AssignmentResult{}, err
	}
	oldByRelay := map[string]model.RelayAssignment{}
	for _, row := range oldRows {
		oldByRelay[row.RelayNodeID] = row
	}
	relayRows, err := s.repo.Relays(ctx, true)
	if err != nil {
		return AssignmentResult{}, err
	}
	relayByID := map[string]model.RelayNode{}
	for _, relay := range relayRows {
		if relay.CancelledAt == nil {
			relayByID[relay.NodeID] = relay
		}
	}
	deviceRows, err := s.repo.Devices(ctx)
	if err != nil {
		return AssignmentResult{}, err
	}
	deviceByID := indexDevices(deviceRows)
	for relayID := range desired {
		relay, exists := relayByID[relayID]
		if !exists {
			return AssignmentResult{}, fmt.Errorf("%w: Relay 不存在", repository.ErrNotFound)
		}
		device, exists := deviceByID[relay.NodeID]
		if !exists || device.Missing {
			return AssignmentResult{}, fmt.Errorf("%w: Relay 设备已删除或无法确认", ErrConflict)
		}
		_, existingAssignment := oldByRelay[relayID]
		if !relay.Selectable && !existingAssignment {
			return AssignmentResult{}, fmt.Errorf("%w: 已暂停的 Relay 不能新增授权", ErrConflict)
		}
		if _, err := selectorIP(selector(device)); err != nil {
			return AssignmentResult{}, fmt.Errorf("%w: Relay 缺少可用于 Policy 的 Tailscale IP", ErrValidation)
		}
	}

	grants := append([]json.RawMessage(nil), doc.Grants...)
	newRows := make([]model.RelayAssignment, 0, len(desired))
	added, removed := 0, 0
	now := s.now().UTC()
	for _, row := range oldRows {
		if desired[row.RelayNodeID] {
			continue
		}
		if row.Managed {
			count, countErr := countExact(grants, row.SourceSelector, row.RelaySelector)
			if countErr != nil {
				return AssignmentResult{}, countErr
			}
			if count > 1 {
				return AssignmentResult{}, fmt.Errorf("%w: 同一授权在 Policy 中出现多次，不能确定删除哪一条", ErrConflict)
			}
			if count == 1 {
				grants, err = removeOneExact(grants, row.SourceSelector, row.RelaySelector)
				if err != nil {
					return AssignmentResult{}, err
				}
				removed++
			}
		}
	}
	for relayID := range desired {
		relayDevice := deviceByID[relayID]
		sourceSelector, _ := selectorIP(selector(source))
		relaySelector, _ := selectorIP(selector(relayDevice))
		old, hadOld := oldByRelay[relayID]
		if hadOld && old.Managed {
			count, countErr := countExact(grants, old.SourceSelector, old.RelaySelector)
			if countErr != nil {
				return AssignmentResult{}, countErr
			}
			if count > 1 {
				return AssignmentResult{}, fmt.Errorf("%w: 同一授权在 Policy 中出现多次", ErrConflict)
			}
			if count == 1 && old.SourceSelector == sourceSelector && old.RelaySelector == relaySelector {
				old.UpdatedBy, old.UpdatedAt = username, now
				newRows = append(newRows, old)
				continue
			}
			if count == 1 {
				grants, err = removeOneExact(grants, old.SourceSelector, old.RelaySelector)
				if err != nil {
					return AssignmentResult{}, err
				}
				removed++
			}
		}
		count, countErr := countExact(grants, sourceSelector, relaySelector)
		if countErr != nil {
			return AssignmentResult{}, countErr
		}
		if count > 1 {
			return AssignmentResult{}, fmt.Errorf("%w: 目标授权在 Policy 中重复", ErrConflict)
		}
		managed := false
		if count == 0 {
			exact, inherited, unknown, matchErr := doc.EffectiveRelay(selector(source), selector(relayDevice))
			if matchErr != nil {
				return AssignmentResult{}, matchErr
			}
			if !exact && !inherited && !unknown {
				grant, marshalErr := policy.NewRelayGrant(sourceSelector, relaySelector)
				if marshalErr != nil {
					return AssignmentResult{}, marshalErr
				}
				grants = append(grants, grant)
				managed = true
				added++
			}
		}
		createdAt := now
		if hadOld {
			createdAt = old.CreatedAt
		}
		newRows = append(newRows, model.RelayAssignment{SourceNodeID: input.SourceNodeID, RelayNodeID: relayID, SourceSelector: sourceSelector, RelaySelector: relaySelector, Managed: managed, CreatedBy: func() string {
			if hadOld {
				return old.CreatedBy
			}
			return username
		}(), UpdatedBy: username, CreatedAt: createdAt, UpdatedAt: now})
	}
	policyChanged := added+removed > 0
	if policyChanged {
		if !doc.HasGrants() {
			return AssignmentResult{}, fmt.Errorf("%w: 当前 Policy 没有 grants 数组，系统拒绝在可能改变默认访问行为的情况下自动创建", ErrConflict)
		}
		newPolicy, err := doc.ReplaceGrants(grants)
		if err != nil {
			return AssignmentResult{}, err
		}
		updatedDoc, parseErr := policy.Parse(newPolicy)
		if parseErr != nil {
			return AssignmentResult{}, parseErr
		}
		beforeHash, afterHash := doc.Hash(), updatedDoc.Hash()
		revision := model.PolicyRevision{ID: uuid.NewString(), Actor: username, Summary: fmt.Sprintf("Peer Relay 授权新增 %d 条、移除 %d 条", added, removed), Before: doc.Raw, After: newPolicy, BeforeHash: beforeHash, AfterHash: afterHash, Status: "pending", CreatedAt: now}
		if err := s.repo.CreateRevisionWithAudit(ctx, revision, s.auditRecord(username, "policy.commit", revision.ID, jsonString(map[string]string{"hash": beforeHash}), jsonString(map[string]string{"hash": afterHash}), "pending")); err != nil {
			return AssignmentResult{}, err
		}
		// The Headscale API does not provide a compare-and-swap write. Re-read as
		// close to the PUT as possible so edits made during preparation are not
		// overwritten. External writes can still race this final read.
		latestRaw, latestErr := s.headscale.GetPolicy(ctx)
		if latestErr != nil {
			finishErr := s.repo.FinishRevisionWithAudit(ctx, revision.ID, "", "failed", s.auditRecord(username, "policy.commit", revision.ID, beforeHash, afterHash, "failed"))
			if finishErr != nil {
				return AssignmentResult{}, finishErr
			}
			return AssignmentResult{}, fmt.Errorf("%w: 写入前无法再次读取最新 Policy", ErrUnavailable)
		}
		latestDoc, parseLatestErr := policy.Parse(latestRaw)
		if parseLatestErr != nil || latestDoc.Hash() != beforeHash {
			finishErr := s.repo.FinishRevisionWithAudit(ctx, revision.ID, "", "failed", s.auditRecord(username, "policy.commit", revision.ID, beforeHash, afterHash, "conflict"))
			if finishErr != nil {
				return AssignmentResult{}, finishErr
			}
			return AssignmentResult{}, fmt.Errorf("%w: Headscale Policy 在提交前再次发生变化", ErrConflict)
		}
		setErr := s.headscale.SetPolicy(ctx, newPolicy)
		readBack, readErr := s.headscale.GetPolicy(ctx)
		readDoc, parseErr := policy.Parse(readBack)
		if readErr != nil || parseErr != nil || readDoc.Hash() != afterHash {
			finishErr := s.repo.FinishRevisionWithAudit(ctx, revision.ID, "", "failed", s.auditRecord(username, "policy.commit", revision.ID, beforeHash, afterHash, "failed"))
			if finishErr != nil {
				return AssignmentResult{}, finishErr
			}
			if setErr != nil {
				return AssignmentResult{}, fmt.Errorf("%w: Headscale 未确认 Policy 提交结果", ErrUnavailable)
			}
			return AssignmentResult{}, fmt.Errorf("%w: Policy 回读结果与提交草稿不同", ErrConflict)
		}
		// A matching read-back remains authoritative if SetPolicy returned a timeout.
		newPolicy = readBack
		if err := s.repo.CompletePolicyUpdate(ctx, revision.ID, []string{input.SourceNodeID}, newPolicy, newRows, []model.AuditLog{
			s.auditRecord(username, "policy.commit", revision.ID, beforeHash, afterHash, "success"),
			s.auditRecord(username, "assignment.update", input.SourceNodeID, jsonString(oldRows), jsonString(newRows), "success"),
		}); err != nil {
			return AssignmentResult{}, err
		}
		_ = s.repo.TrimRevisions(ctx, 30)
		state, _ := s.repo.SyncState(ctx)
		state.Key, state.PolicyHash, state.PolicyJSON, state.LastSuccess = "headscale", afterHash, newPolicy, &now
		state.LastError = ""
		_ = s.repo.SaveSyncState(ctx, state)
	}
	if !policyChanged {
		if err := s.repo.SaveAssignmentsWithAudit(ctx, input.SourceNodeID, newRows, s.auditRecord(username, "assignment.update", input.SourceNodeID, jsonString(oldRows), jsonString(newRows), "success")); err != nil {
			return AssignmentResult{}, err
		}
	}
	graph, err := s.Graph(ctx, username, input.SourceNodeID)
	if err != nil {
		return AssignmentResult{}, err
	}
	return AssignmentResult{Graph: graph, Added: added, Removed: removed, PolicyChanged: policyChanged}, nil
}

func (s *PeerRelayService) UpdateAssignmentsBatch(ctx context.Context, username string, input BatchAssignmentInput) (BatchAssignmentResult, error) {
	if len(input.Changes) == 0 || len(input.Changes) > 500 || len(input.BasePolicyHash) != 64 {
		return BatchAssignmentResult{}, ErrValidation
	}
	changes := append([]AssignmentChange(nil), input.Changes...)
	sort.Slice(changes, func(i, j int) bool { return changes[i].SourceNodeID < changes[j].SourceNodeID })
	for i := range changes {
		if changes[i].SourceNodeID == "" || len(changes[i].SourceNodeID) > 120 {
			return BatchAssignmentResult{}, ErrValidation
		}
		if i > 0 && changes[i-1].SourceNodeID == changes[i].SourceNodeID {
			return BatchAssignmentResult{}, ErrValidation
		}
		seen := make(map[string]bool, len(changes[i].RelayNodeIDs))
		for _, relayID := range changes[i].RelayNodeIDs {
			if relayID == "" || len(relayID) > 120 || seen[relayID] {
				return BatchAssignmentResult{}, ErrValidation
			}
			seen[relayID] = true
		}
	}

	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	doc, _, role, err := s.prepareWrite(ctx, username)
	if err != nil {
		return BatchAssignmentResult{}, err
	}
	if !s.cfg.PolicyWrites {
		return BatchAssignmentResult{}, ErrPolicyWrites
	}
	if doc.Hash() != input.BasePolicyHash {
		return BatchAssignmentResult{}, fmt.Errorf("%w: Headscale Policy 已更新，请重新载入画布后再保存", ErrConflict)
	}

	deviceRows, err := s.repo.Devices(ctx)
	if err != nil {
		return BatchAssignmentResult{}, err
	}
	deviceByID := indexDevices(deviceRows)
	relayRows, err := s.repo.Relays(ctx, true)
	if err != nil {
		return BatchAssignmentResult{}, err
	}
	relayByID := make(map[string]model.RelayNode, len(relayRows))
	for _, relay := range relayRows {
		if relay.CancelledAt == nil {
			relayByID[relay.NodeID] = relay
		}
	}
	allRows, err := s.repo.Assignments(ctx, "")
	if err != nil {
		return BatchAssignmentResult{}, err
	}
	rowsBySource := make(map[string][]model.RelayAssignment)
	for _, row := range allRows {
		rowsBySource[row.SourceNodeID] = append(rowsBySource[row.SourceNodeID], row)
	}

	desiredBySource := make(map[string]map[string]bool, len(changes))
	oldBySource := make(map[string]map[string]model.RelayAssignment, len(changes))
	sources := make(map[string]model.Device, len(changes))
	for _, change := range changes {
		source, exists := deviceByID[change.SourceNodeID]
		if !exists || source.Missing {
			return BatchAssignmentResult{}, repository.ErrNotFound
		}
		if !role.canEditAllDevices && source.UserID != role.user.ID {
			return BatchAssignmentResult{}, ErrForbidden
		}
		if len(deviceView(source).Tags) > 0 && !role.isAdmin {
			return BatchAssignmentResult{}, ErrForbidden
		}
		if !role.isAdmin && !role.canEditAllDevices && !role.selfServiceEnabled {
			return BatchAssignmentResult{}, ErrForbidden
		}
		if _, ipErr := selectorIP(selector(source)); ipErr != nil {
			return BatchAssignmentResult{}, fmt.Errorf("%w: 源设备缺少可用于 Policy 的 Tailscale IP", ErrValidation)
		}
		sources[change.SourceNodeID] = source
		desired := make(map[string]bool, len(change.RelayNodeIDs))
		for _, relayID := range change.RelayNodeIDs {
			if relayID == change.SourceNodeID {
				return BatchAssignmentResult{}, ErrValidation
			}
			desired[relayID] = true
		}
		desiredBySource[change.SourceNodeID] = desired
		oldByRelay := make(map[string]model.RelayAssignment)
		for _, row := range rowsBySource[change.SourceNodeID] {
			oldByRelay[row.RelayNodeID] = row
		}
		oldBySource[change.SourceNodeID] = oldByRelay

		for relayID := range desired {
			relay, registered := relayByID[relayID]
			if !registered {
				return BatchAssignmentResult{}, fmt.Errorf("%w: 目标设备不是当前注册的 Peer Relay", ErrConflict)
			}
			relayDevice, exists := deviceByID[relay.NodeID]
			if !exists || relayDevice.Missing {
				if _, hadAssignment := oldByRelay[relayID]; !hadAssignment {
					return BatchAssignmentResult{}, fmt.Errorf("%w: Relay 设备已删除或无法确认", ErrConflict)
				}
				continue
			}
			_, hadAssignment := oldByRelay[relayID]
			exact, inherited, unknown, relationErr := doc.EffectiveRelay(selector(source), selector(relayDevice))
			if relationErr != nil {
				return BatchAssignmentResult{}, fmt.Errorf("%w: 无法确认目标授权关系", ErrConflict)
			}
			if !relay.Selectable && !hadAssignment && !exact && !inherited && !unknown {
				return BatchAssignmentResult{}, fmt.Errorf("%w: 已暂停的 Relay 不能新增授权", ErrConflict)
			}
			if _, ipErr := selectorIP(selector(relayDevice)); ipErr != nil && !hadAssignment && !exact && !inherited && !unknown {
				return BatchAssignmentResult{}, fmt.Errorf("%w: Relay 缺少可用于 Policy 的 Tailscale IP", ErrValidation)
			}
		}

		// A managed edge whose endpoint is missing or no longer has a usable IP
		// cannot be safely removed from the complete Policy document.
		for relayID, old := range oldByRelay {
			if desired[relayID] || !old.Managed {
				continue
			}
			relayDevice, exists := deviceByID[relayID]
			if !exists || relayDevice.Missing {
				return BatchAssignmentResult{}, fmt.Errorf("%w: 无法确认的既有授权不能移除", ErrConflict)
			}
			if _, ipErr := selectorIP(selector(relayDevice)); ipErr != nil {
				return BatchAssignmentResult{}, fmt.Errorf("%w: 无法确认的既有授权不能移除", ErrConflict)
			}
		}
	}

	grants := append([]json.RawMessage(nil), doc.Grants...)
	newRowsBySource := make(map[string][]model.RelayAssignment, len(changes))
	added, removed := 0, 0
	now := s.now().UTC()
	for _, change := range changes {
		source := sources[change.SourceNodeID]
		desired := desiredBySource[change.SourceNodeID]
		oldByRelay := oldBySource[change.SourceNodeID]
		for relayID, old := range oldByRelay {
			if desired[relayID] || !old.Managed {
				continue
			}
			count, countErr := countExact(grants, old.SourceSelector, old.RelaySelector)
			if countErr != nil {
				return BatchAssignmentResult{}, countErr
			}
			if count > 1 {
				return BatchAssignmentResult{}, fmt.Errorf("%w: 同一授权在 Policy 中出现多次，不能确定删除哪一条", ErrConflict)
			}
			if count == 1 {
				grants, err = removeOneExact(grants, old.SourceSelector, old.RelaySelector)
				if err != nil {
					return BatchAssignmentResult{}, err
				}
				removed++
			}
		}

		relayIDs := make([]string, 0, len(desired))
		for relayID := range desired {
			relayIDs = append(relayIDs, relayID)
		}
		sort.Strings(relayIDs)
		newRows := make([]model.RelayAssignment, 0, len(desired))
		sourceSelector, _ := selectorIP(selector(source))
		for _, relayID := range relayIDs {
			relay := relayByID[relayID]
			relayDevice, exists := deviceByID[relay.NodeID]
			old, hadOld := oldByRelay[relayID]
			if !exists || relayDevice.Missing {
				if hadOld {
					newRows = append(newRows, old)
					continue
				}
				return BatchAssignmentResult{}, fmt.Errorf("%w: Relay 设备已删除或无法确认", ErrConflict)
			}
			relaySelector, relayIPErr := selectorIP(selector(relayDevice))
			if relayIPErr != nil {
				if hadOld {
					newRows = append(newRows, old)
					continue
				}
				continue
			}
			if hadOld && old.Managed {
				oldCount, countErr := countExact(grants, old.SourceSelector, old.RelaySelector)
				if countErr != nil {
					return BatchAssignmentResult{}, countErr
				}
				if oldCount > 1 {
					return BatchAssignmentResult{}, fmt.Errorf("%w: 同一授权在 Policy 中出现多次", ErrConflict)
				}
				if oldCount == 1 && old.SourceSelector == sourceSelector && old.RelaySelector == relaySelector {
					old.UpdatedBy, old.UpdatedAt = username, now
					newRows = append(newRows, old)
					continue
				}
				if oldCount == 0 {
					currentCount, currentCountErr := countExact(grants, sourceSelector, relaySelector)
					if currentCountErr != nil {
						return BatchAssignmentResult{}, currentCountErr
					}
					if currentCount > 1 {
						return BatchAssignmentResult{}, fmt.Errorf("%w: 目标授权在 Policy 中重复", ErrConflict)
					}
					if currentCount == 0 {
						// Retain the visible exception without silently repairing Policy.
						newRows = append(newRows, old)
						continue
					}
				}
				if oldCount == 1 {
					grants, err = removeOneExact(grants, old.SourceSelector, old.RelaySelector)
					if err != nil {
						return BatchAssignmentResult{}, err
					}
					removed++
				}
			}
			count, countErr := countExact(grants, sourceSelector, relaySelector)
			if countErr != nil {
				return BatchAssignmentResult{}, countErr
			}
			if count > 1 {
				return BatchAssignmentResult{}, fmt.Errorf("%w: 目标授权在 Policy 中重复", ErrConflict)
			}
			managed := false
			if count == 0 {
				exact, inherited, unknown, matchErr := doc.EffectiveRelay(selector(source), selector(relayDevice))
				if matchErr != nil {
					return BatchAssignmentResult{}, matchErr
				}
				if !exact && !inherited && !unknown {
					grant, marshalErr := policy.NewRelayGrant(sourceSelector, relaySelector)
					if marshalErr != nil {
						return BatchAssignmentResult{}, marshalErr
					}
					grants = append(grants, grant)
					managed = true
					added++
				}
			}
			createdAt := now
			createdBy := username
			if hadOld {
				createdAt, createdBy = old.CreatedAt, old.CreatedBy
			}
			newRows = append(newRows, model.RelayAssignment{
				SourceNodeID: change.SourceNodeID, RelayNodeID: relayID,
				SourceSelector: sourceSelector, RelaySelector: relaySelector, Managed: managed,
				CreatedBy: createdBy, UpdatedBy: username, CreatedAt: createdAt, UpdatedAt: now,
			})
		}
		newRowsBySource[change.SourceNodeID] = newRows
	}

	newPolicy := doc.Raw
	policyChanged := added+removed > 0
	sourceIDs := make([]string, 0, len(changes))
	newRows := make([]model.RelayAssignment, 0)
	for _, change := range changes {
		sourceIDs = append(sourceIDs, change.SourceNodeID)
		newRows = append(newRows, newRowsBySource[change.SourceNodeID]...)
	}
	changeAudit := s.auditRecord(username, "assignment.batch_update", "multiple-devices", "", jsonString(map[string]any{
		"sourceNodeIds": sourceIDs, "added": added, "removed": removed,
	}), "success")
	if policyChanged {
		if !doc.HasGrants() {
			return BatchAssignmentResult{}, fmt.Errorf("%w: 当前 Policy 没有 grants 数组，系统拒绝在可能改变默认访问行为的情况下自动创建", ErrConflict)
		}
		newPolicy, err = doc.ReplaceGrants(grants)
		if err != nil {
			return BatchAssignmentResult{}, err
		}
		updatedDoc, parseErr := policy.Parse(newPolicy)
		if parseErr != nil {
			return BatchAssignmentResult{}, parseErr
		}
		beforeHash, afterHash := doc.Hash(), updatedDoc.Hash()
		revision := model.PolicyRevision{
			ID: uuid.NewString(), Actor: username,
			Summary: fmt.Sprintf("Peer Relay 批量授权：%d 台设备，新增 %d 条、移除 %d 条", len(changes), added, removed),
			Before:  doc.Raw, After: newPolicy, BeforeHash: beforeHash, AfterHash: afterHash,
			Status: "pending", CreatedAt: now,
		}
		if err := s.repo.CreateRevisionWithAudit(ctx, revision, s.auditRecord(username, "policy.commit", revision.ID, jsonString(map[string]string{"hash": beforeHash}), jsonString(map[string]string{"hash": afterHash}), "pending")); err != nil {
			return BatchAssignmentResult{}, err
		}
		latestRaw, latestErr := s.headscale.GetPolicy(ctx)
		if latestErr != nil {
			finishErr := s.repo.FinishRevisionWithAudit(ctx, revision.ID, "", "failed", s.auditRecord(username, "policy.commit", revision.ID, beforeHash, afterHash, "failed"))
			if finishErr != nil {
				return BatchAssignmentResult{}, finishErr
			}
			return BatchAssignmentResult{}, fmt.Errorf("%w: 写入前无法再次读取最新 Policy", ErrUnavailable)
		}
		latestDoc, parseLatestErr := policy.Parse(latestRaw)
		if parseLatestErr != nil || latestDoc.Hash() != beforeHash {
			finishErr := s.repo.FinishRevisionWithAudit(ctx, revision.ID, "", "failed", s.auditRecord(username, "policy.commit", revision.ID, beforeHash, afterHash, "conflict"))
			if finishErr != nil {
				return BatchAssignmentResult{}, finishErr
			}
			return BatchAssignmentResult{}, fmt.Errorf("%w: Headscale Policy 在提交前再次发生变化", ErrConflict)
		}
		setErr := s.headscale.SetPolicy(ctx, newPolicy)
		readBack, readErr := s.headscale.GetPolicy(ctx)
		readDoc, parseErr := policy.Parse(readBack)
		if readErr != nil || parseErr != nil || readDoc.Hash() != afterHash {
			finishErr := s.repo.FinishRevisionWithAudit(ctx, revision.ID, "", "failed", s.auditRecord(username, "policy.commit", revision.ID, beforeHash, afterHash, "failed"))
			if finishErr != nil {
				return BatchAssignmentResult{}, finishErr
			}
			if setErr != nil {
				return BatchAssignmentResult{}, fmt.Errorf("%w: Headscale 未确认 Policy 提交结果", ErrUnavailable)
			}
			return BatchAssignmentResult{}, fmt.Errorf("%w: Policy 回读结果与提交草稿不同", ErrConflict)
		}
		newPolicy = readBack
		changeAudit.BeforeJSON = jsonString(map[string]string{"hash": beforeHash})
		changeAudit.AfterJSON = jsonString(map[string]any{"hash": afterHash, "sourceNodeIds": sourceIDs, "added": added, "removed": removed})
		if err := s.repo.CompletePolicyUpdate(ctx, revision.ID, sourceIDs, newPolicy, newRows, []model.AuditLog{
			s.auditRecord(username, "policy.commit", revision.ID, beforeHash, afterHash, "success"),
			changeAudit,
		}); err != nil {
			return BatchAssignmentResult{}, err
		}
		_ = s.repo.TrimRevisions(ctx, 30)
		state, _ := s.repo.SyncState(ctx)
		state.Key, state.PolicyHash, state.PolicyJSON, state.LastSuccess = "headscale", afterHash, newPolicy, &now
		state.LastError = ""
		_ = s.repo.SaveSyncState(ctx, state)
	} else {
		if err := s.repo.SaveAssignmentsBatchWithAudit(ctx, sourceIDs, newRows, changeAudit); err != nil {
			return BatchAssignmentResult{}, err
		}
	}
	finalHash := doc.Hash()
	if policyChanged {
		updatedDoc, parseErr := policy.Parse(newPolicy)
		if parseErr != nil {
			return BatchAssignmentResult{}, parseErr
		}
		finalHash = updatedDoc.Hash()
	}
	return BatchAssignmentResult{
		BasePolicyHash: finalHash, ChangedSourceCount: len(changes), Added: added,
		Removed: removed, PolicyChanged: policyChanged, SourceNodeIDs: sourceIDs,
	}, nil
}

func (s *PeerRelayService) PreviewSelectorRule(ctx context.Context, username string, input SelectorRulePreviewInput) (SelectorRulePreview, error) {
	if strings.TrimSpace(input.SourceSelector) == "" || strings.TrimSpace(input.RelayNodeID) == "" || len(input.RelayNodeID) > 120 {
		return SelectorRulePreview{}, ErrValidation
	}
	if err := s.Sync(ctx); err != nil {
		return SelectorRulePreview{}, err
	}
	doc, _, readErr := s.readPolicy(ctx)
	if readErr != nil || doc == nil {
		return SelectorRulePreview{}, fmt.Errorf("%w: 无法读取当前 Headscale Policy", ErrUnavailable)
	}
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return SelectorRulePreview{}, err
	}
	if !role.isAdmin {
		return SelectorRulePreview{}, ErrForbidden
	}
	return s.selectorRulePreview(ctx, doc, input)
}

func (s *PeerRelayService) SelectorPolicyRules(ctx context.Context, username string) ([]SelectorRuleView, error) {
	if err := s.Sync(ctx); err != nil {
		return nil, err
	}
	doc, _, readErr := s.readPolicy(ctx)
	if readErr != nil || doc == nil {
		return nil, fmt.Errorf("%w: 无法读取当前 Headscale Policy", ErrUnavailable)
	}
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return nil, err
	}
	if !role.isAdmin {
		return nil, ErrForbidden
	}
	rules, err := s.repo.SelectorPolicyRules(ctx)
	if err != nil {
		return nil, err
	}
	users, err := s.repo.Users(ctx)
	if err != nil {
		return nil, err
	}
	devices, err := s.repo.Devices(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]SelectorRuleView, 0, len(rules))
	for _, rule := range rules {
		view := SelectorRuleView{
			ID: rule.ID, SourceSelector: rule.SourceSelector, RelayNodeID: rule.RelayNodeID,
			RelaySelector: rule.RelaySelector, SourceDevices: []DeviceView{}, CreatedBy: rule.CreatedBy, CreatedAt: rule.CreatedAt,
		}
		if err := s.fillSelectorRuleSources(doc, users, devices, rule.SourceSelector, &view.SourceDevices); err != nil {
			view.ReadOnly = true
			view.Warning = err.Error()
		}
		if count, countErr := countExact(doc.Grants, rule.SourceSelector, rule.RelaySelector); countErr != nil {
			view.ReadOnly = true
			appendViewWarning(&view.Warning, "无法安全解析当前 Relay Policy")
		} else if count == 1 {
			view.InPolicy = true
			view.CanDelete = true
		} else if count == 0 {
			complexCount, pairErr := countPeerRelaySelectorPair(doc.Grants, rule.SourceSelector, rule.RelaySelector)
			if pairErr != nil || complexCount > 0 {
				view.ReadOnly = true
				appendViewWarning(&view.Warning, "Policy 中的 selector 规则已被改成复杂形式，Tailor 保持只读")
			} else {
				view.CanDelete = true
				appendViewWarning(&view.Warning, "对应授权已不在 Policy 中；移除只会清理 Tailor 的归属记录")
			}
		} else {
			view.ReadOnly = true
			appendViewWarning(&view.Warning, "同一 selector 授权在 Policy 中重复，无法确定删除对象")
		}
		relay, relayErr := s.repo.Relay(ctx, rule.RelayNodeID)
		if relayErr == nil {
			device, deviceErr := s.repo.Device(ctx, rule.RelayNodeID)
			if deviceErr == nil {
				relayView := RelayView{RelayNode: relay, Device: deviceView(device)}
				view.Relay = &relayView
				currentIP, ipErr := selectorIP(selector(device))
				view.TargetChanged = ipErr != nil || currentIP != rule.RelaySelector
				if view.TargetChanged {
					appendViewWarning(&view.Warning, "已注册 Relay 的当前 IP 与该规则目标不同")
				}
			} else {
				view.TargetChanged = true
				appendViewWarning(&view.Warning, "Relay 设备当前不可用")
			}
		} else {
			view.TargetChanged = true
			appendViewWarning(&view.Warning, "Relay 注册记录当前不可用")
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *PeerRelayService) CreateSelectorRule(ctx context.Context, username string, input SelectorRuleInput) (SelectorRuleMutationResult, error) {
	if len(input.BasePolicyHash) != 64 || len(input.ScopeHash) != 64 {
		return SelectorRuleMutationResult{}, ErrValidation
	}
	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	doc, _, role, err := s.prepareWrite(ctx, username)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	if !role.isAdmin {
		return SelectorRuleMutationResult{}, ErrForbidden
	}
	if !s.cfg.PolicyWrites {
		return SelectorRuleMutationResult{}, ErrPolicyWrites
	}
	if doc.Hash() != input.BasePolicyHash {
		return SelectorRuleMutationResult{}, fmt.Errorf("%w: Headscale Policy 已更新，请重新预览 selector 规则", ErrConflict)
	}
	preview, err := s.selectorRulePreview(ctx, doc, SelectorRulePreviewInput{SourceSelector: input.SourceSelector, RelayNodeID: input.RelayNodeID})
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	if preview.ScopeHash != input.ScopeHash {
		return SelectorRuleMutationResult{}, fmt.Errorf("%w: 当前命中设备已变化，请重新预览授权范围", ErrConflict)
	}
	rules, err := s.repo.SelectorPolicyRules(ctx)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	for _, existing := range rules {
		if strings.EqualFold(existing.SourceSelector, preview.SourceSelector) && existing.RelaySelector == preview.RelaySelector {
			return SelectorRuleMutationResult{}, fmt.Errorf("%w: Tailor 已管理相同的 selector 规则", ErrConflict)
		}
	}
	existingCount, err := countPeerRelaySelectorPair(doc.Grants, preview.SourceSelector, preview.RelaySelector)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	if existingCount > 0 {
		return SelectorRuleMutationResult{}, fmt.Errorf("%w: Policy 中已存在相同 selector 和 Relay 目标，Tailor 不会覆盖或接管该规则", ErrConflict)
	}
	grant, err := policy.NewRelayGrant(preview.SourceSelector, preview.RelaySelector)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	grants := append(append([]json.RawMessage{}, doc.Grants...), grant)
	newPolicy, err := doc.ReplaceGrants(grants)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	now := s.now().UTC()
	rule := model.SelectorPolicyRule{
		ID: uuid.NewString(), SourceSelector: preview.SourceSelector, RelayNodeID: preview.RelayNodeID,
		RelaySelector: preview.RelaySelector, CreatedBy: username, CreatedAt: now, UpdatedAt: now,
	}
	revision, readBack, afterHash, err := s.writeSelectorPolicy(ctx, username, "Peer Relay selector 授权新增："+preview.SourceSelector+" → "+preview.RelaySelector, doc, newPolicy)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	if err := s.repo.CompleteSelectorPolicyUpdate(ctx, revision.ID, readBack, &rule, "", []model.AuditLog{
		s.auditRecord(username, "selector-rule.create", rule.ID, "", jsonString(rule), "success"),
		s.auditRecord(username, "policy.commit", revision.ID, revision.BeforeHash, afterHash, "success"),
	}); err != nil {
		return SelectorRuleMutationResult{}, err
	}
	_ = s.repo.TrimRevisions(ctx, 30)
	s.savePolicySyncState(ctx, readBack, afterHash, now)
	return SelectorRuleMutationResult{ID: rule.ID, SourceSelector: rule.SourceSelector, RelayNodeID: rule.RelayNodeID, RelaySelector: rule.RelaySelector, BasePolicyHash: afterHash}, nil
}

func (s *PeerRelayService) DeleteSelectorRule(ctx context.Context, username, id, basePolicyHash string) (SelectorRuleMutationResult, error) {
	if id == "" || len(id) > 36 || len(basePolicyHash) != 64 {
		return SelectorRuleMutationResult{}, ErrValidation
	}
	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	doc, _, role, err := s.prepareWrite(ctx, username)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	if !role.isAdmin {
		return SelectorRuleMutationResult{}, ErrForbidden
	}
	if doc.Hash() != basePolicyHash {
		return SelectorRuleMutationResult{}, fmt.Errorf("%w: Headscale Policy 已更新，请刷新 selector 规则列表", ErrConflict)
	}
	rule, err := s.repo.SelectorPolicyRule(ctx, id)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	count, err := countExact(doc.Grants, rule.SourceSelector, rule.RelaySelector)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	if count > 1 {
		return SelectorRuleMutationResult{}, fmt.Errorf("%w: 同一 selector 授权在 Policy 中重复，不能确定删除哪一条", ErrConflict)
	}
	if count == 0 {
		complexCount, err := countPeerRelaySelectorPair(doc.Grants, rule.SourceSelector, rule.RelaySelector)
		if err != nil {
			return SelectorRuleMutationResult{}, err
		}
		if complexCount > 0 {
			return SelectorRuleMutationResult{}, fmt.Errorf("%w: Policy 中的 selector 规则已被改成复杂形式，Tailor 保持只读", ErrConflict)
		}
		if err := s.repo.DeleteSelectorPolicyRuleWithAudit(ctx, id, s.auditRecord(username, "selector-rule.orphan.delete", id, jsonString(rule), "", "success")); err != nil {
			return SelectorRuleMutationResult{}, err
		}
		return SelectorRuleMutationResult{ID: id, SourceSelector: rule.SourceSelector, RelayNodeID: rule.RelayNodeID, RelaySelector: rule.RelaySelector, BasePolicyHash: doc.Hash()}, nil
	}
	if !s.cfg.PolicyWrites {
		return SelectorRuleMutationResult{}, ErrPolicyWrites
	}
	grants, err := removeOneExact(doc.Grants, rule.SourceSelector, rule.RelaySelector)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	newPolicy, err := doc.ReplaceGrants(grants)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	now := s.now().UTC()
	revision, readBack, afterHash, err := s.writeSelectorPolicy(ctx, username, "Peer Relay selector 授权移除："+rule.SourceSelector+" → "+rule.RelaySelector, doc, newPolicy)
	if err != nil {
		return SelectorRuleMutationResult{}, err
	}
	if err := s.repo.CompleteSelectorPolicyUpdate(ctx, revision.ID, readBack, nil, id, []model.AuditLog{
		s.auditRecord(username, "selector-rule.delete", id, jsonString(rule), "", "success"),
		s.auditRecord(username, "policy.commit", revision.ID, revision.BeforeHash, afterHash, "success"),
	}); err != nil {
		return SelectorRuleMutationResult{}, err
	}
	_ = s.repo.TrimRevisions(ctx, 30)
	s.savePolicySyncState(ctx, readBack, afterHash, now)
	return SelectorRuleMutationResult{ID: id, SourceSelector: rule.SourceSelector, RelayNodeID: rule.RelayNodeID, RelaySelector: rule.RelaySelector, BasePolicyHash: afterHash}, nil
}

func (s *PeerRelayService) selectorRulePreview(ctx context.Context, doc *policy.Document, input SelectorRulePreviewInput) (SelectorRulePreview, error) {
	users, err := s.repo.Users(ctx)
	if err != nil {
		return SelectorRulePreview{}, err
	}
	sourceSelector, err := validateWritableSourceSelector(input.SourceSelector, doc, users)
	if err != nil {
		return SelectorRulePreview{}, err
	}
	relay, err := s.repo.Relay(ctx, input.RelayNodeID)
	if err != nil || relay.CancelledAt != nil || !relay.Selectable {
		return SelectorRulePreview{}, fmt.Errorf("%w: 目标必须是已注册且可选择的 Relay", ErrValidation)
	}
	device, err := s.repo.Device(ctx, relay.NodeID)
	if err != nil || device.Missing {
		return SelectorRulePreview{}, fmt.Errorf("%w: Relay 设备当前不可用", ErrValidation)
	}
	relaySelector, err := selectorIP(selector(device))
	if err != nil {
		return SelectorRulePreview{}, fmt.Errorf("%w: Relay 缺少可用于 Policy 的 Tailscale IPv4 地址", ErrValidation)
	}
	devices, err := s.repo.Devices(ctx)
	if err != nil {
		return SelectorRulePreview{}, err
	}
	sources := make([]DeviceView, 0)
	for _, candidate := range devices {
		if candidate.Missing {
			continue
		}
		matched, known := doc.MatchSourceSelector(sourceSelector, selector(candidate))
		if !known {
			return SelectorRulePreview{}, fmt.Errorf("%w: 无法完整评估 selector %s", ErrValidation, sourceSelector)
		}
		if matched {
			sources = append(sources, deviceView(candidate))
		}
	}
	preview := SelectorRulePreview{
		SourceSelector: sourceSelector, RelayNodeID: relay.NodeID, RelaySelector: relaySelector,
		Relay: RelayView{RelayNode: relay, Device: deviceView(device)}, SourceDevices: sources,
		BasePolicyHash: doc.Hash(), Dynamic: true,
	}
	preview.ScopeHash = selectorRuleScopeHash(sourceSelector, relay.NodeID, relaySelector, sources)
	if len(sources) == 0 {
		preview.Warning = "当前没有匹配设备；用户设备或 Group 成员后续变化时，授权范围会动态更新"
	} else {
		preview.Warning = "用户设备或 Group 成员后续变化时，授权范围会动态更新"
	}
	return preview, nil
}

func (s *PeerRelayService) fillSelectorRuleSources(doc *policy.Document, users []model.HeadscaleUser, devices []model.Device, rawSelector string, result *[]DeviceView) error {
	sourceSelector, err := validateWritableSourceSelector(rawSelector, doc, users)
	if err != nil {
		return err
	}
	for _, device := range devices {
		if device.Missing {
			continue
		}
		matched, known := doc.MatchSourceSelector(sourceSelector, selector(device))
		if !known {
			return fmt.Errorf("Policy selector %s 无法完整评估", sourceSelector)
		}
		if matched {
			*result = append(*result, deviceView(device))
		}
	}
	return nil
}

func (s *PeerRelayService) writeSelectorPolicy(ctx context.Context, actor, summary string, before *policy.Document, after string) (model.PolicyRevision, string, string, error) {
	updated, err := policy.Parse(after)
	if err != nil {
		return model.PolicyRevision{}, "", "", err
	}
	beforeHash, afterHash := before.Hash(), updated.Hash()
	revision := model.PolicyRevision{
		ID: uuid.NewString(), Actor: actor, Summary: summary, Before: before.Raw, After: after,
		BeforeHash: beforeHash, AfterHash: afterHash, Status: "pending", CreatedAt: s.now().UTC(),
	}
	if err := s.repo.CreateRevisionWithAudit(ctx, revision, s.auditRecord(actor, "policy.commit", revision.ID, jsonString(map[string]string{"hash": beforeHash}), jsonString(map[string]string{"hash": afterHash}), "pending")); err != nil {
		return model.PolicyRevision{}, "", "", err
	}
	finish := func(status string) error {
		return s.repo.FinishRevisionWithAudit(ctx, revision.ID, "", status, s.auditRecord(actor, "policy.commit", revision.ID, beforeHash, afterHash, status))
	}
	latestRaw, err := s.headscale.GetPolicy(ctx)
	if err != nil {
		if finishErr := finish("failed"); finishErr != nil {
			return model.PolicyRevision{}, "", "", finishErr
		}
		return model.PolicyRevision{}, "", "", fmt.Errorf("%w: 写入前无法再次读取最新 Policy", ErrUnavailable)
	}
	latest, err := policy.Parse(latestRaw)
	if err != nil || latest.Hash() != beforeHash {
		if finishErr := finish("conflict"); finishErr != nil {
			return model.PolicyRevision{}, "", "", finishErr
		}
		return model.PolicyRevision{}, "", "", fmt.Errorf("%w: Headscale Policy 在提交前再次发生变化", ErrConflict)
	}
	setErr := s.headscale.SetPolicy(ctx, after)
	readBack, readErr := s.headscale.GetPolicy(ctx)
	readDoc, parseErr := policy.Parse(readBack)
	if readErr != nil || parseErr != nil || readDoc.Hash() != afterHash {
		status := "failed"
		if finishErr := finish(status); finishErr != nil {
			return model.PolicyRevision{}, "", "", finishErr
		}
		if setErr != nil || readErr != nil {
			return model.PolicyRevision{}, "", "", fmt.Errorf("%w: Headscale 未确认 Policy 提交结果", ErrUnavailable)
		}
		return model.PolicyRevision{}, "", "", fmt.Errorf("%w: Policy 回读结果与提交草稿不同", ErrConflict)
	}
	return revision, readBack, afterHash, nil
}

func (s *PeerRelayService) savePolicySyncState(ctx context.Context, raw, hash string, now time.Time) {
	state, _ := s.repo.SyncState(ctx)
	state.Key, state.PolicyHash, state.PolicyJSON, state.LastSuccess = "headscale", hash, raw, &now
	state.LastError = ""
	_ = s.repo.SaveSyncState(ctx, state)
}

func countPeerRelaySelectorPair(grants []json.RawMessage, source, relay string) (int, error) {
	count := 0
	for _, raw := range grants {
		grant, err := policy.ParseGrant(raw)
		if err != nil {
			return 0, err
		}
		if grant.PeerRelay && len(grant.Src) == 1 && len(grant.Dst) == 1 && strings.EqualFold(grant.Src[0], source) && grant.Dst[0] == relay {
			count++
		}
	}
	return count, nil
}

func appendViewWarning(target *string, warning string) {
	if warning == "" {
		return
	}
	if *target == "" {
		*target = warning
		return
	}
	*target += "；" + warning
}

func (s *PeerRelayService) Revisions(ctx context.Context, username string) ([]RevisionView, error) {
	doc, _, _ := s.readPolicy(ctx)
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return nil, err
	}
	if !role.isAdmin {
		return nil, ErrForbidden
	}
	revisions, err := s.repo.Revisions(ctx, 30)
	if err != nil {
		return nil, err
	}
	result := make([]RevisionView, 0, len(revisions))
	for _, revision := range revisions {
		result = append(result, revisionView(revision))
	}
	return result, nil
}

func (s *PeerRelayService) RevisionDetail(ctx context.Context, username, id string) (RevisionDetailView, error) {
	doc, _, _ := s.readPolicy(ctx)
	var err error
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return RevisionDetailView{}, err
	}
	if !role.isAdmin {
		return RevisionDetailView{}, ErrForbidden
	}
	revision, err := s.repo.Revision(ctx, id)
	if err != nil {
		return RevisionDetailView{}, err
	}
	detail := RevisionDetailView{RevisionView: revisionView(revision), Before: revision.Before, After: revision.After, AddedRelayGrants: []RelayGrantChange{}, RemovedRelayGrants: []RelayGrantChange{}}
	var before, after *policy.Document
	if revision.Before != "" {
		before, err = policy.Parse(revision.Before)
		if err != nil {
			return RevisionDetailView{}, fmt.Errorf("%w: Policy 版本快照无法解析", ErrConflict)
		}
	}
	if revision.After != "" {
		after, err = policy.Parse(revision.After)
		if err != nil {
			return RevisionDetailView{}, fmt.Errorf("%w: Policy 版本快照无法解析", ErrConflict)
		}
	}
	if before == nil || after == nil {
		return detail, nil
	}
	beforeGrants, beforeErr := before.PeerRelayGrants()
	afterGrants, afterErr := after.PeerRelayGrants()
	if beforeErr != nil || afterErr != nil {
		return RevisionDetailView{}, fmt.Errorf("%w: Peer Relay Grants 无法解析", ErrConflict)
	}
	beforeByKey := make(map[string]RelayGrantChange, len(beforeGrants))
	afterByKey := make(map[string]RelayGrantChange, len(afterGrants))
	for _, grant := range beforeGrants {
		change := relayGrantChange(grant.Src, grant.Dst)
		beforeByKey[relayGrantChangeKey(change)] = change
	}
	for _, grant := range afterGrants {
		change := relayGrantChange(grant.Src, grant.Dst)
		afterByKey[relayGrantChangeKey(change)] = change
	}
	for key, grant := range afterByKey {
		if _, exists := beforeByKey[key]; !exists {
			detail.AddedRelayGrants = append(detail.AddedRelayGrants, grant)
		}
	}
	for key, grant := range beforeByKey {
		if _, exists := afterByKey[key]; !exists {
			detail.RemovedRelayGrants = append(detail.RemovedRelayGrants, grant)
		}
	}
	sort.Slice(detail.AddedRelayGrants, func(i, j int) bool {
		return relayGrantChangeKey(detail.AddedRelayGrants[i]) < relayGrantChangeKey(detail.AddedRelayGrants[j])
	})
	sort.Slice(detail.RemovedRelayGrants, func(i, j int) bool {
		return relayGrantChangeKey(detail.RemovedRelayGrants[i]) < relayGrantChangeKey(detail.RemovedRelayGrants[j])
	})
	beforeOther, err := policyWithoutRelayGrants(before)
	if err != nil {
		return RevisionDetailView{}, fmt.Errorf("%w: Policy 其他字段无法比较", ErrConflict)
	}
	afterOther, err := policyWithoutRelayGrants(after)
	if err != nil {
		return RevisionDetailView{}, fmt.Errorf("%w: Policy 其他字段无法比较", ErrConflict)
	}
	detail.OtherPolicyChanged = beforeOther != afterOther
	return detail, nil
}

func relayGrantChange(src, dst []string) RelayGrantChange {
	return RelayGrantChange{Src: append([]string(nil), src...), Dst: append([]string(nil), dst...)}
}

func relayGrantChangeKey(value RelayGrantChange) string {
	source := append([]string(nil), value.Src...)
	destination := append([]string(nil), value.Dst...)
	sort.Strings(source)
	sort.Strings(destination)
	return jsonString(struct {
		Src []string `json:"src"`
		Dst []string `json:"dst"`
	}{Src: source, Dst: destination})
}

func policyWithoutRelayGrants(doc *policy.Document) (string, error) {
	copy := make(map[string]json.RawMessage, len(doc.Root))
	for key, value := range doc.Root {
		if key != "grants" {
			copy[key] = value
		}
	}
	if doc.HasGrants() {
		otherGrants := make([]json.RawMessage, 0)
		for _, raw := range doc.Grants {
			grant, err := policy.ParseGrant(raw)
			if err != nil {
				return "", err
			}
			if !grant.PeerRelay {
				otherGrants = append(otherGrants, raw)
			}
		}
		encoded, err := json.Marshal(otherGrants)
		if err != nil {
			return "", err
		}
		copy["grants"] = encoded
	}
	encoded, err := json.Marshal(copy)
	return string(encoded), err
}

func (s *PeerRelayService) Rollback(ctx context.Context, username, id string) (RevisionView, error) {
	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	doc, _, role, err := s.prepareWrite(ctx, username)
	if err != nil {
		return RevisionView{}, err
	}
	if !role.isAdmin {
		return RevisionView{}, ErrForbidden
	}
	if !s.cfg.PolicyWrites {
		return RevisionView{}, ErrPolicyWrites
	}
	revision, err := s.repo.Revision(ctx, id)
	if err != nil {
		return RevisionView{}, err
	}
	if revision.Status != "success" {
		return RevisionView{}, fmt.Errorf("%w: 只能回滚成功提交的版本", ErrConflict)
	}
	latest, err := s.repo.LatestSuccessfulRevision(ctx)
	if err != nil {
		return RevisionView{}, err
	}
	if latest.ID == "" || doc.Hash() != latest.AfterHash {
		return RevisionView{}, fmt.Errorf("%w: 当前 Policy 与最近一次平台提交不一致，可能已有外部修改", ErrConflict)
	}
	previous, err := policy.Parse(revision.Before)
	if err != nil {
		return RevisionView{}, err
	}
	rollback := model.PolicyRevision{ID: uuid.NewString(), Actor: username, Summary: "回滚 Policy 版本 " + id, Before: doc.Raw, After: previous.Raw, BeforeHash: doc.Hash(), AfterHash: previous.Hash(), Status: "pending", CreatedAt: s.now().UTC()}
	if err := s.repo.CreateRevisionWithAudit(ctx, rollback, s.auditRecord(username, "policy.rollback", id, jsonString(map[string]string{"hash": rollback.BeforeHash}), jsonString(map[string]string{"hash": rollback.AfterHash}), "pending")); err != nil {
		return RevisionView{}, err
	}
	latestRaw, latestErr := s.headscale.GetPolicy(ctx)
	if latestErr != nil {
		finishErr := s.repo.FinishRevisionWithAudit(ctx, rollback.ID, "", "failed", s.auditRecord(username, "policy.rollback", id, doc.Hash(), previous.Hash(), "failed"))
		if finishErr != nil {
			return RevisionView{}, finishErr
		}
		return RevisionView{}, fmt.Errorf("%w: 回滚前无法再次读取最新 Policy", ErrUnavailable)
	}
	latestDoc, parseLatestErr := policy.Parse(latestRaw)
	if parseLatestErr != nil || latestDoc.Hash() != doc.Hash() {
		finishErr := s.repo.FinishRevisionWithAudit(ctx, rollback.ID, "", "failed", s.auditRecord(username, "policy.rollback", id, doc.Hash(), previous.Hash(), "conflict"))
		if finishErr != nil {
			return RevisionView{}, finishErr
		}
		return RevisionView{}, fmt.Errorf("%w: Headscale Policy 在回滚提交前发生变化", ErrConflict)
	}
	setErr := s.headscale.SetPolicy(ctx, previous.Raw)
	readBack, readErr := s.headscale.GetPolicy(ctx)
	readDoc, parseErr := policy.Parse(readBack)
	if readErr != nil || parseErr != nil || readDoc.Hash() != previous.Hash() {
		finishErr := s.repo.FinishRevisionWithAudit(ctx, rollback.ID, "", "failed", s.auditRecord(username, "policy.rollback", id, doc.Hash(), previous.Hash(), "failed"))
		if finishErr != nil {
			return RevisionView{}, finishErr
		}
		return RevisionView{}, fmt.Errorf("%w: 回滚结果未能回读确认", ErrUnavailable)
	}
	_ = setErr
	if err := s.repo.FinishRevisionWithAudit(ctx, rollback.ID, readBack, "success", s.auditRecord(username, "policy.rollback", id, doc.Hash(), previous.Hash(), "success")); err != nil {
		return RevisionView{}, err
	}
	_ = s.repo.TrimRevisions(ctx, 30)
	state, _ := s.repo.SyncState(ctx)
	now := s.now().UTC()
	state.Key, state.PolicyHash, state.PolicyJSON, state.LastSuccess = "headscale", readDoc.Hash(), readBack, &now
	state.LastError = ""
	_ = s.repo.SaveSyncState(ctx, state)
	rollback.Status, rollback.After = "success", readBack
	return revisionView(rollback), nil
}

func (s *PeerRelayService) Audits(ctx context.Context, username string) ([]AuditView, error) {
	doc, _, _ := s.readPolicy(ctx)
	role, err := s.roles(ctx, username, doc)
	if err != nil {
		return nil, err
	}
	if !role.isAdmin {
		return nil, ErrForbidden
	}
	values, err := s.repo.Audits(ctx, 100)
	if err != nil {
		return nil, err
	}
	result := make([]AuditView, 0, len(values))
	for _, value := range values {
		result = append(result, AuditView{ID: value.ID, Actor: value.Actor, Action: value.Action, Target: value.Target, Before: value.BeforeJSON, After: value.AfterJSON, Result: value.Result, CreatedAt: value.CreatedAt})
	}
	return result, nil
}

func (s *PeerRelayService) readPolicy(ctx context.Context) (*policy.Document, bool, error) {
	if s.headscale.Configured() {
		if raw, err := s.headscale.GetPolicy(ctx); err == nil {
			parsed, parseErr := policy.Parse(raw)
			return parsed, false, parseErr
		}
	}
	state, err := s.repo.SyncState(ctx)
	if err != nil || state.PolicyJSON == "" {
		return nil, true, ErrUnavailable
	}
	parsed, parseErr := policy.Parse(state.PolicyJSON)
	return parsed, true, parseErr
}

func (s *PeerRelayService) prepareWrite(ctx context.Context, username string) (*policy.Document, bool, roleContext, error) {
	if err := s.Sync(ctx); err != nil {
		return nil, false, roleContext{}, err
	}
	raw, err := s.headscale.GetPolicy(ctx)
	if err != nil {
		return nil, false, roleContext{}, fmt.Errorf("%w: 无法读取最新 Policy", ErrUnavailable)
	}
	doc, err := policy.Parse(raw)
	if err != nil {
		return nil, false, roleContext{}, err
	}
	role, err := s.roles(ctx, username, doc)
	return doc, false, role, err
}

func (s *PeerRelayService) roles(ctx context.Context, username string, doc *policy.Document) (roleContext, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return roleContext{}, ErrForbidden
	}
	users, err := s.repo.Users(ctx)
	if err != nil {
		return roleContext{}, err
	}
	var user model.HeadscaleUser
	for _, candidate := range users {
		if strings.EqualFold(candidate.Username, username) {
			user = candidate
			break
		}
	}
	if user.ID == "" {
		return roleContext{}, ErrForbidden
	}
	settings, err := s.accessSettings(ctx)
	if err != nil {
		return roleContext{}, err
	}
	adminGroups, reviewerGroups, applicantGroups, err := decodeSettingsGroups(settings)
	if err != nil {
		return roleContext{}, err
	}
	role := roleContext{user: user, groups: []string{}}
	for group, members := range policyGroups(doc) {
		if groupMember(username, user.Email, members) {
			role.groups = append(role.groups, group)
		}
	}
	sort.Strings(role.groups)
	role.isAdmin = stringContainsFold(s.cfg.BootstrapAdmins, username) || belongsToAny(role.groups, adminGroups)
	role.reviewer = role.isAdmin || belongsToAny(role.groups, reviewerGroups)
	role.canManageRelays = role.isAdmin || (role.reviewer && settings.ReviewerManageRelays)
	role.canEditAllDevices = role.isAdmin || (role.reviewer && settings.ReviewerEditAllDevices)
	role.selfServiceEnabled = settings.SelfServiceEnabled
	switch settings.ApplicantMode {
	case "all":
		role.canApply = true
	case "groups":
		role.canApply = belongsToAny(role.groups, applicantGroups)
	}
	return role, nil
}

func (s *PeerRelayService) userView(role roleContext) UserView {
	return UserView{Username: role.user.Username, Email: role.user.Email, Groups: role.groups, IsAdmin: role.isAdmin, IsReviewer: role.reviewer, CanApply: role.canApply, CanManageRelays: role.canManageRelays, CanEditAllDevices: role.canEditAllDevices}
}

func policyGroups(doc *policy.Document) map[string][]string {
	if doc == nil {
		return map[string][]string{}
	}
	return doc.Groups
}

func groupMember(username, email string, members []string) bool {
	for _, member := range members {
		member = strings.TrimSpace(member)
		if strings.EqualFold(member, username) || (email != "" && strings.EqualFold(member, email)) || strings.EqualFold(strings.TrimSuffix(member, "@"), strings.TrimSuffix(username, "@")) {
			return true
		}
		if !strings.Contains(username, "@") && strings.EqualFold(strings.TrimSuffix(member, "@"), username) {
			return true
		}
	}
	return false
}

func validateWritableSourceSelector(raw string, doc *policy.Document, users []model.HeadscaleUser) (string, error) {
	selector := strings.TrimSpace(raw)
	if selector == "" || len([]rune(selector)) > 255 || strings.ContainsAny(selector, ",[]*\n\r\t") {
		return "", ErrValidation
	}
	if strings.HasPrefix(selector, "group:") {
		if _, ok := doc.Groups[selector]; !ok {
			return "", fmt.Errorf("%w: Policy 中不存在 Group %s", ErrValidation, selector)
		}
		if err := validateWritableGroup(selector, doc.Groups, users, map[string]bool{}, 0); err != nil {
			return "", err
		}
		return selector, nil
	}
	if strings.HasPrefix(selector, "tag:") || strings.HasPrefix(selector, "autogroup:") || strings.Contains(selector, "/") || strings.Contains(selector, ":") {
		return "", ErrValidation
	}
	if len(matchingPolicyUsers(selector, users)) != 1 {
		return "", fmt.Errorf("%w: 用户 selector 必须唯一匹配一个当前 Headscale 用户", ErrValidation)
	}
	return selector, nil
}

func validateWritableGroup(group string, groups map[string][]string, users []model.HeadscaleUser, seen map[string]bool, depth int) error {
	if depth > 8 || seen[group] {
		return fmt.Errorf("%w: Group 嵌套存在循环或超过支持深度", ErrValidation)
	}
	members, ok := groups[group]
	if !ok {
		return fmt.Errorf("%w: Policy 中不存在嵌套 Group %s", ErrValidation, group)
	}
	seen[group] = true
	defer delete(seen, group)
	for _, member := range members {
		member = strings.TrimSpace(member)
		if strings.HasPrefix(member, "group:") {
			if err := validateWritableGroup(member, groups, users, seen, depth+1); err != nil {
				return err
			}
			continue
		}
		if member == "" || len(matchingPolicyUsers(member, users)) != 1 {
			return fmt.Errorf("%w: Group %s 含有无法完整识别的成员", ErrValidation, group)
		}
	}
	return nil
}

func matchingPolicyUsers(selector string, users []model.HeadscaleUser) []model.HeadscaleUser {
	matches := make([]model.HeadscaleUser, 0, 1)
	for _, user := range users {
		username := strings.TrimSuffix(strings.TrimSpace(user.Username), "@")
		if strings.EqualFold(selector, user.Username) || strings.EqualFold(selector, username+"@") || (user.Email != "" && strings.EqualFold(selector, user.Email)) {
			matches = append(matches, user)
		}
	}
	return matches
}

func selectorRuleScopeHash(selector, relayID, relayIP string, sources []DeviceView) string {
	nodeIDs := make([]string, 0, len(sources))
	for _, source := range sources {
		nodeIDs = append(nodeIDs, source.NodeID)
	}
	sort.Strings(nodeIDs)
	canonical := selector + "\x00" + relayID + "\x00" + relayIP + "\x00" + strings.Join(nodeIDs, "\x00")
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

func belongsToAny(groups, allowed []string) bool {
	for _, group := range groups {
		if stringContainsFold(allowed, group) {
			return true
		}
	}
	return false
}

func stringContainsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func deviceView(value model.Device) DeviceView {
	var tags []string
	_ = json.Unmarshal([]byte(value.TagsJSON), &tags)
	if tags == nil {
		tags = []string{}
	}
	return DeviceView{NodeID: value.NodeID, UserID: value.UserID, Hostname: value.Hostname, GivenName: value.GivenName, IPv4: value.IPv4, IPv6: value.IPv6, Username: value.Username, UserEmail: value.UserEmail, Tags: tags, Online: value.Online, LastSeen: value.LastSeen, Expiry: value.Expiry, Missing: value.Missing}
}

func selector(value model.Device) policy.DeviceSelector {
	view := deviceView(value)
	ips := []string{}
	if view.IPv4 != "" {
		ips = append(ips, view.IPv4)
	}
	if view.IPv6 != "" {
		ips = append(ips, view.IPv6)
	}
	return policy.DeviceSelector{UserID: value.UserID, Username: view.Username, Email: value.UserEmail, Hostname: view.Hostname, GivenName: view.GivenName, Tags: view.Tags, IPs: ips, Tagged: len(view.Tags) > 0}
}

func selectorFromView(value DeviceView) policy.DeviceSelector {
	ips := []string{}
	if value.IPv4 != "" {
		ips = append(ips, value.IPv4)
	}
	if value.IPv6 != "" {
		ips = append(ips, value.IPv6)
	}
	return policy.DeviceSelector{UserID: value.UserID, Username: value.Username, Email: value.UserEmail, Hostname: value.Hostname, GivenName: value.GivenName, Tags: value.Tags, IPs: ips, Tagged: len(value.Tags) > 0}
}

func selectorIP(device policy.DeviceSelector) (string, error) {
	for _, raw := range device.IPs {
		address, err := netip.ParseAddr(raw)
		if err == nil && address.Is4() {
			return address.String(), nil
		}
	}
	for _, raw := range device.IPs {
		address, err := netip.ParseAddr(raw)
		if err == nil && address.Is6() {
			return address.String(), nil
		}
	}
	return "", ErrValidation
}

func countExact(grants []json.RawMessage, source, relay string) (int, error) {
	count := 0
	for _, raw := range grants {
		grant, err := policy.ParseGrant(raw)
		if err != nil {
			return 0, err
		}
		if grant.ExactShape && grant.Src[0] == source && grant.Dst[0] == relay {
			count++
		}
	}
	return count, nil
}

func removeOneExact(grants []json.RawMessage, source, relay string) ([]json.RawMessage, error) {
	for i, raw := range grants {
		grant, err := policy.ParseGrant(raw)
		if err != nil {
			return nil, err
		}
		if grant.ExactShape && grant.Src[0] == source && grant.Dst[0] == relay {
			return append(grants[:i:i], grants[i+1:]...), nil
		}
	}
	return grants, nil
}

func indexDevices(devices []model.Device) map[string]model.Device {
	result := make(map[string]model.Device, len(devices))
	for _, device := range devices {
		result[device.NodeID] = device
	}
	return result
}

func tailnetIPs(values []string) (string, string) {
	ipv4, ipv6 := "", ""
	for _, raw := range values {
		address, err := netip.ParseAddr(strings.TrimSpace(raw))
		if err != nil {
			continue
		}
		if address.Is4() && ipv4 == "" {
			ipv4 = address.String()
		}
		if address.Is6() && ipv6 == "" {
			ipv6 = address.String()
		}
	}
	return ipv4, ipv6
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func normalizeRelayInput(input RelayInput) (RelayInput, error) {
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Region, input.PublicEndpoint = strings.TrimSpace(input.Region), strings.TrimSpace(input.PublicEndpoint)
	input.ISP, input.Description = strings.TrimSpace(input.ISP), strings.TrimSpace(input.Description)
	if input.DisplayName == "" || len([]rune(input.DisplayName)) > 80 || len([]rune(input.Region)) > 120 || len([]rune(input.ISP)) > 120 || len([]rune(input.Description)) > 2000 {
		return RelayInput{}, ErrValidation
	}
	if input.UDPPort != nil && (*input.UDPPort < 1 || *input.UDPPort > 65535) {
		return RelayInput{}, ErrValidation
	}
	if err := validateEndpoint(input.PublicEndpoint); err != nil {
		return RelayInput{}, ErrValidation
	}
	return input, nil
}

func normalizeApplicationInput(input ApplicationInput) (ApplicationInput, error) {
	input.NodeID = strings.TrimSpace(input.NodeID)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Region, input.PublicEndpoint = strings.TrimSpace(input.Region), strings.TrimSpace(input.PublicEndpoint)
	input.ISP, input.Reason = strings.TrimSpace(input.ISP), strings.TrimSpace(input.Reason)
	if input.NodeID == "" || input.DisplayName == "" || len([]rune(input.DisplayName)) > 80 || len([]rune(input.Region)) > 120 || len([]rune(input.ISP)) > 120 || len([]rune(input.Reason)) > 2000 {
		return ApplicationInput{}, ErrValidation
	}
	if input.UDPPort != nil && (*input.UDPPort < 1 || *input.UDPPort > 65535) {
		return ApplicationInput{}, ErrValidation
	}
	if err := validateEndpoint(input.PublicEndpoint); err != nil {
		return ApplicationInput{}, ErrValidation
	}
	return input, nil
}

func validateEndpoint(value string) error {
	if value == "" {
		return nil
	}
	if net.ParseIP(value) != nil {
		return nil
	}
	if strings.ContainsAny(value, "/?#@ ") || strings.Contains(value, ":") {
		return errors.New("invalid endpoint")
	}
	parsed, err := url.Parse("https://" + value)
	if err != nil || parsed.Host != value || len(value) > 253 {
		return errors.New("invalid endpoint")
	}
	for _, label := range strings.Split(value, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return errors.New("invalid endpoint")
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && char != '-' {
				return errors.New("invalid endpoint")
			}
		}
	}
	return nil
}

func (s *PeerRelayService) audit(ctx context.Context, actor, action, target, before, after, result string) error {
	return s.repo.Audit(ctx, s.auditRecord(actor, action, target, before, after, result))
}

func (s *PeerRelayService) auditRecord(actor, action, target, before, after, result string) model.AuditLog {
	return model.AuditLog{Actor: actor, Action: action, Target: target, BeforeJSON: before, AfterJSON: after, Result: result, CreatedAt: s.now().UTC()}
}

func jsonString(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func revisionView(value model.PolicyRevision) RevisionView {
	return RevisionView{ID: value.ID, Actor: value.Actor, Summary: value.Summary, BeforeHash: value.BeforeHash, AfterHash: value.AfterHash, Status: value.Status, CreatedAt: value.CreatedAt}
}
