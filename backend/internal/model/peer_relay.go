package model

import "time"

type HeadscaleUser struct {
	ID       string    `gorm:"primaryKey;size:120" json:"id"`
	Username string    `gorm:"size:255;not null;index" json:"username"`
	Email    string    `gorm:"size:255" json:"email,omitempty"`
	SyncedAt time.Time `gorm:"not null" json:"syncedAt"`
}

type Device struct {
	NodeID    string     `gorm:"primaryKey;size:120" json:"nodeId"`
	UserID    string     `gorm:"size:120;index" json:"userId"`
	Username  string     `gorm:"size:255;not null;index" json:"username"`
	UserEmail string     `gorm:"size:255" json:"-"`
	Hostname  string     `gorm:"size:255;not null" json:"hostname"`
	GivenName string     `gorm:"size:255" json:"givenName"`
	IPv4      string     `gorm:"size:64;index" json:"ipv4,omitempty"`
	IPv6      string     `gorm:"size:128" json:"ipv6,omitempty"`
	TagsJSON  string     `gorm:"type:text" json:"-"`
	Online    *bool      `json:"online,omitempty"`
	LastSeen  *time.Time `json:"lastSeen,omitempty"`
	Expiry    *time.Time `json:"expiry,omitempty"`
	Missing   bool       `gorm:"not null;default:false;index" json:"missing"`
	SyncedAt  time.Time  `gorm:"not null;index" json:"syncedAt"`
}

func (Device) TableName() string { return "devices" }

type RelayNode struct {
	NodeID         string     `gorm:"primaryKey;size:120" json:"nodeId"`
	DisplayName    string     `gorm:"size:255;not null" json:"displayName"`
	Region         string     `gorm:"size:120" json:"region,omitempty"`
	UDPPort        *int       `json:"udpPort,omitempty"`
	PublicEndpoint string     `gorm:"size:255" json:"publicEndpoint,omitempty"`
	ISP            string     `gorm:"size:255" json:"isp,omitempty"`
	Description    string     `gorm:"type:text" json:"description,omitempty"`
	Selectable     bool       `gorm:"not null;default:true;index" json:"selectable"`
	CancelledAt    *time.Time `json:"cancelledAt,omitempty"`
	CreatedBy      string     `gorm:"size:255;not null" json:"createdBy"`
	CreatedAt      time.Time  `gorm:"not null" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"not null" json:"updatedAt"`
}

func (RelayNode) TableName() string { return "relay_nodes" }

type RelayApplication struct {
	ID              string     `gorm:"primaryKey;size:36" json:"id"`
	Applicant       string     `gorm:"size:255;not null;index" json:"applicant"`
	ApplicantUserID string     `gorm:"size:120;not null;index" json:"-"`
	NodeID          string     `gorm:"size:120;not null;index;uniqueIndex:idx_pending_application_node,where:status = 'pending'" json:"nodeId"`
	DisplayName     string     `gorm:"size:255;not null" json:"displayName"`
	Region          string     `gorm:"size:120" json:"region,omitempty"`
	UDPPort         *int       `json:"udpPort,omitempty"`
	PublicEndpoint  string     `gorm:"size:255" json:"publicEndpoint,omitempty"`
	ISP             string     `gorm:"size:255" json:"isp,omitempty"`
	Reason          string     `gorm:"type:text" json:"reason,omitempty"`
	Status          string     `gorm:"size:24;not null;index" json:"status"`
	Reviewer        string     `gorm:"size:255" json:"reviewer,omitempty"`
	ReviewNote      string     `gorm:"type:text" json:"reviewNote,omitempty"`
	CreatedAt       time.Time  `gorm:"not null;index" json:"createdAt"`
	ReviewedAt      *time.Time `json:"reviewedAt,omitempty"`
	UpdatedAt       time.Time  `gorm:"not null" json:"updatedAt"`
}

func (RelayApplication) TableName() string { return "relay_applications" }

type RelayAssignment struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	SourceNodeID   string    `gorm:"size:120;not null;uniqueIndex:idx_relay_assignment_pair" json:"sourceNodeId"`
	RelayNodeID    string    `gorm:"size:120;not null;uniqueIndex:idx_relay_assignment_pair" json:"relayNodeId"`
	SourceSelector string    `gorm:"size:128;not null" json:"-"`
	RelaySelector  string    `gorm:"size:128;not null" json:"-"`
	Managed        bool      `gorm:"not null;default:false" json:"managed"`
	CreatedBy      string    `gorm:"size:255;not null" json:"createdBy"`
	UpdatedBy      string    `gorm:"size:255;not null" json:"updatedBy"`
	CreatedAt      time.Time `gorm:"not null" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"not null" json:"updatedAt"`
}

func (RelayAssignment) TableName() string { return "relay_assignments" }

type PolicyRevision struct {
	ID         string    `gorm:"primaryKey;size:36" json:"id"`
	Actor      string    `gorm:"size:255;not null;index" json:"actor"`
	Summary    string    `gorm:"size:1000;not null" json:"summary"`
	Before     string    `gorm:"type:text;not null" json:"-"`
	After      string    `gorm:"type:text;not null" json:"-"`
	BeforeHash string    `gorm:"size:64;not null" json:"beforeHash"`
	AfterHash  string    `gorm:"size:64;not null" json:"afterHash"`
	Status     string    `gorm:"size:24;not null;index" json:"status"`
	CreatedAt  time.Time `gorm:"not null;index" json:"createdAt"`
}

func (PolicyRevision) TableName() string { return "policy_revisions" }

type AuditLog struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Actor      string    `gorm:"size:255;not null;index" json:"actor"`
	Action     string    `gorm:"size:80;not null;index" json:"action"`
	Target     string    `gorm:"size:255;not null;index" json:"target"`
	BeforeJSON string    `gorm:"type:text" json:"before,omitempty"`
	AfterJSON  string    `gorm:"type:text" json:"after,omitempty"`
	Result     string    `gorm:"size:24;not null" json:"result"`
	CreatedAt  time.Time `gorm:"not null;index" json:"createdAt"`
}

func (AuditLog) TableName() string { return "audit_logs" }

type SyncState struct {
	Key         string     `gorm:"primaryKey;size:32"`
	LastAttempt *time.Time `json:"lastAttempt,omitempty"`
	LastSuccess *time.Time `json:"lastSuccess,omitempty"`
	LastError   string     `gorm:"type:text" json:"lastError,omitempty"`
	PolicyHash  string     `gorm:"size:64" json:"policyHash,omitempty"`
	PolicyJSON  string     `gorm:"type:text" json:"-"`
}

func (SyncState) TableName() string { return "sync_state" }

type PeerRelaySettings struct {
	ID                     uint      `gorm:"primaryKey" json:"-"`
	AdminGroupsJSON        string    `gorm:"type:text;not null" json:"-"`
	ReviewerGroupsJSON     string    `gorm:"type:text;not null" json:"-"`
	ApplicantGroupsJSON    string    `gorm:"type:text;not null" json:"-"`
	ApplicantMode          string    `gorm:"size:16;not null" json:"applicantMode"`
	ReviewerManageRelays   bool      `gorm:"not null;default:false" json:"reviewerManageRelays"`
	ReviewerEditAllDevices bool      `gorm:"not null;default:false" json:"reviewerEditAllDevices"`
	SelfServiceEnabled     bool      `gorm:"not null;default:true" json:"selfServiceEnabled"`
	SyncIntervalSeconds    int       `gorm:"not null;default:60" json:"syncIntervalSeconds"`
	ReapplyCooldownHours   int       `gorm:"not null;default:0" json:"reapplyCooldownHours"`
	UpdatedBy              string    `gorm:"size:255;not null" json:"updatedBy"`
	UpdatedAt              time.Time `gorm:"not null" json:"updatedAt"`
}

func (PeerRelaySettings) TableName() string { return "peer_relay_settings" }
