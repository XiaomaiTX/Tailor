package repository

import (
	"context"
	"errors"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"gorm.io/gorm"
)

var ErrApplicationNotPending = errors.New("申请不再处于待审批状态")

type PeerRelayRepository interface {
	SyncSnapshot(context.Context, []model.HeadscaleUser, []model.Device) error
	Users(context.Context) ([]model.HeadscaleUser, error)
	Devices(context.Context) ([]model.Device, error)
	Device(context.Context, string) (model.Device, error)
	Relays(context.Context, bool) ([]model.RelayNode, error)
	Relay(context.Context, string) (model.RelayNode, error)
	SaveRelayWithAudit(context.Context, model.RelayNode, model.AuditLog) error
	Applications(context.Context, string, string, string) ([]model.RelayApplication, error)
	Application(context.Context, string) (model.RelayApplication, error)
	PendingApplication(context.Context, string) (model.RelayApplication, error)
	LastRejectedApplication(context.Context, string) (model.RelayApplication, error)
	CreateApplicationWithAudit(context.Context, model.RelayApplication, model.AuditLog) error
	CancelPendingApplicationWithAudit(context.Context, string, time.Time, model.AuditLog) error
	InvalidatePendingApplicationWithAudit(context.Context, model.RelayApplication, model.AuditLog) error
	CompleteApplicationReview(context.Context, model.RelayApplication, *model.RelayNode, model.AuditLog) error
	Assignments(context.Context, string) ([]model.RelayAssignment, error)
	SelectorPolicyRules(context.Context) ([]model.SelectorPolicyRule, error)
	SelectorPolicyRule(context.Context, string) (model.SelectorPolicyRule, error)
	SaveAssignmentsWithAudit(context.Context, string, []model.RelayAssignment, model.AuditLog) error
	SaveAssignmentsBatchWithAudit(context.Context, []string, []model.RelayAssignment, model.AuditLog) error
	CreateRevisionWithAudit(context.Context, model.PolicyRevision, model.AuditLog) error
	FinishRevisionWithAudit(context.Context, string, string, string, model.AuditLog) error
	CompletePolicyUpdate(context.Context, string, []string, string, []model.RelayAssignment, []model.AuditLog) error
	CompleteSelectorPolicyUpdate(context.Context, string, string, *model.SelectorPolicyRule, string, []model.AuditLog) error
	DeleteSelectorPolicyRuleWithAudit(context.Context, string, model.AuditLog) error
	Revisions(context.Context, int) ([]model.PolicyRevision, error)
	Revision(context.Context, string) (model.PolicyRevision, error)
	LatestSuccessfulRevision(context.Context) (model.PolicyRevision, error)
	TrimRevisions(context.Context, int) error
	Audit(context.Context, model.AuditLog) error
	Audits(context.Context, int) ([]model.AuditLog, error)
	SaveSyncState(context.Context, model.SyncState) error
	SyncState(context.Context) (model.SyncState, error)
	PeerRelaySettings(context.Context) (model.PeerRelaySettings, error)
	SavePeerRelaySettingsWithAudit(context.Context, model.PeerRelaySettings, model.AuditLog) error
}

type peerRelayRepository struct{ db *gorm.DB }

func NewPeerRelayRepository(db *gorm.DB) PeerRelayRepository {
	return &peerRelayRepository{db: db}
}

func (r *peerRelayRepository) SyncSnapshot(ctx context.Context, users []model.HeadscaleUser, devices []model.Device) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, user := range users {
			if err := tx.Save(&user).Error; err != nil {
				return err
			}
		}
		userIDs := make([]string, 0, len(users))
		for _, user := range users {
			userIDs = append(userIDs, user.ID)
		}
		userQuery := tx.Model(&model.HeadscaleUser{})
		if len(userIDs) > 0 {
			userQuery = userQuery.Where("id NOT IN ?", userIDs)
		}
		if err := userQuery.Delete(&model.HeadscaleUser{}).Error; err != nil {
			return err
		}
		for _, device := range devices {
			if err := tx.Save(&device).Error; err != nil {
				return err
			}
		}
		ids := make([]string, 0, len(devices))
		for _, device := range devices {
			ids = append(ids, device.NodeID)
		}
		query := tx.Model(&model.Device{}).Where("missing = ?", false)
		if len(ids) > 0 {
			query = query.Where("node_id NOT IN ?", ids)
		}
		return query.Update("missing", true).Error
	})
}

func (r *peerRelayRepository) Users(ctx context.Context) ([]model.HeadscaleUser, error) {
	var values []model.HeadscaleUser
	err := r.db.WithContext(ctx).Order("username ASC").Find(&values).Error
	return values, err
}

func (r *peerRelayRepository) Devices(ctx context.Context) ([]model.Device, error) {
	var values []model.Device
	err := r.db.WithContext(ctx).Order("username ASC, hostname ASC, node_id ASC").Find(&values).Error
	return values, err
}

func (r *peerRelayRepository) Device(ctx context.Context, nodeID string) (model.Device, error) {
	var value model.Device
	err := r.db.WithContext(ctx).First(&value, "node_id = ?", nodeID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Device{}, ErrNotFound
	}
	return value, err
}

func (r *peerRelayRepository) Relays(ctx context.Context, includeCancelled bool) ([]model.RelayNode, error) {
	query := r.db.WithContext(ctx).Model(&model.RelayNode{}).Order("created_at DESC, node_id ASC")
	if !includeCancelled {
		query = query.Where("cancelled_at IS NULL")
	}
	var values []model.RelayNode
	err := query.Find(&values).Error
	return values, err
}

func (r *peerRelayRepository) Relay(ctx context.Context, nodeID string) (model.RelayNode, error) {
	var value model.RelayNode
	err := r.db.WithContext(ctx).First(&value, "node_id = ?", nodeID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.RelayNode{}, ErrNotFound
	}
	return value, err
}

func (r *peerRelayRepository) SaveRelayWithAudit(ctx context.Context, value model.RelayNode, audit model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&value).Error; err != nil {
			return err
		}
		return tx.Create(&audit).Error
	})
}

func (r *peerRelayRepository) Applications(ctx context.Context, status, applicant, applicantUserID string) ([]model.RelayApplication, error) {
	query := r.db.WithContext(ctx).Model(&model.RelayApplication{}).Order("created_at DESC, id DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if applicant != "" && applicantUserID != "" {
		query = query.Where("applicant = ? OR applicant_user_id = ?", applicant, applicantUserID)
	} else if applicantUserID != "" {
		query = query.Where("applicant_user_id = ?", applicantUserID)
	} else if applicant != "" {
		query = query.Where("applicant = ?", applicant)
	}
	var values []model.RelayApplication
	err := query.Find(&values).Error
	return values, err
}

func (r *peerRelayRepository) Application(ctx context.Context, id string) (model.RelayApplication, error) {
	var value model.RelayApplication
	err := r.db.WithContext(ctx).First(&value, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.RelayApplication{}, ErrNotFound
	}
	return value, err
}

func (r *peerRelayRepository) PendingApplication(ctx context.Context, nodeID string) (model.RelayApplication, error) {
	var value model.RelayApplication
	err := r.db.WithContext(ctx).First(&value, "node_id = ? AND status = ?", nodeID, "pending").Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.RelayApplication{}, ErrNotFound
	}
	return value, err
}

func (r *peerRelayRepository) LastRejectedApplication(ctx context.Context, nodeID string) (model.RelayApplication, error) {
	var value model.RelayApplication
	err := r.db.WithContext(ctx).Where("node_id = ? AND status = ?", nodeID, "rejected").Order("updated_at DESC, id DESC").First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.RelayApplication{}, ErrNotFound
	}
	return value, err
}

func (r *peerRelayRepository) CreateApplicationWithAudit(ctx context.Context, value model.RelayApplication, audit model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&value).Error; err != nil {
			return err
		}
		return tx.Create(&audit).Error
	})
}

func (r *peerRelayRepository) CancelPendingApplicationWithAudit(ctx context.Context, id string, updatedAt time.Time, audit model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.RelayApplication{}).Where("id = ? AND status = ?", id, "pending").Updates(map[string]any{
			"status": "cancelled", "updated_at": updatedAt,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrApplicationNotPending
		}
		return tx.Create(&audit).Error
	})
}

func (r *peerRelayRepository) InvalidatePendingApplicationWithAudit(ctx context.Context, application model.RelayApplication, audit model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.RelayApplication{}).Where("id = ? AND status = ?", application.ID, "pending").Updates(map[string]any{
			"status": application.Status, "review_note": application.ReviewNote, "updated_at": application.UpdatedAt,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		return tx.Create(&audit).Error
	})
}

func (r *peerRelayRepository) CompleteApplicationReview(ctx context.Context, application model.RelayApplication, relay *model.RelayNode, audit model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if relay != nil {
			if err := tx.Create(relay).Error; err != nil {
				return err
			}
		}
		result := tx.Model(&model.RelayApplication{}).Where("id = ? AND status = ?", application.ID, "pending").Updates(map[string]any{
			"status": application.Status, "reviewer": application.Reviewer, "review_note": application.ReviewNote,
			"reviewed_at": application.ReviewedAt, "updated_at": application.UpdatedAt,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrApplicationNotPending
		}
		return tx.Create(&audit).Error
	})
}

func (r *peerRelayRepository) Assignments(ctx context.Context, sourceNodeID string) ([]model.RelayAssignment, error) {
	query := r.db.WithContext(ctx).Model(&model.RelayAssignment{}).Order("relay_node_id ASC")
	if sourceNodeID != "" {
		query = query.Where("source_node_id = ?", sourceNodeID)
	}
	var values []model.RelayAssignment
	err := query.Find(&values).Error
	return values, err
}

func (r *peerRelayRepository) SelectorPolicyRules(ctx context.Context) ([]model.SelectorPolicyRule, error) {
	var values []model.SelectorPolicyRule
	err := r.db.WithContext(ctx).Order("created_at ASC, id ASC").Find(&values).Error
	return values, err
}

func (r *peerRelayRepository) SelectorPolicyRule(ctx context.Context, id string) (model.SelectorPolicyRule, error) {
	var value model.SelectorPolicyRule
	err := r.db.WithContext(ctx).First(&value, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SelectorPolicyRule{}, ErrNotFound
	}
	return value, err
}

func (r *peerRelayRepository) SaveAssignmentsWithAudit(ctx context.Context, sourceNodeID string, values []model.RelayAssignment, audit model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("source_node_id = ?", sourceNodeID).Delete(&model.RelayAssignment{}).Error; err != nil {
			return err
		}
		if len(values) > 0 {
			if err := tx.Create(&values).Error; err != nil {
				return err
			}
		}
		return tx.Create(&audit).Error
	})
}

func (r *peerRelayRepository) SaveAssignmentsBatchWithAudit(ctx context.Context, sourceNodeIDs []string, values []model.RelayAssignment, audit model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(sourceNodeIDs) > 0 {
			if err := tx.Where("source_node_id IN ?", sourceNodeIDs).Delete(&model.RelayAssignment{}).Error; err != nil {
				return err
			}
		}
		if len(values) > 0 {
			if err := tx.Create(&values).Error; err != nil {
				return err
			}
		}
		return tx.Create(&audit).Error
	})
}

func (r *peerRelayRepository) CreateRevisionWithAudit(ctx context.Context, value model.PolicyRevision, audit model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&value).Error; err != nil {
			return err
		}
		return tx.Create(&audit).Error
	})
}

func (r *peerRelayRepository) FinishRevisionWithAudit(ctx context.Context, id, after, status string, audit model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"status": status}
		if after != "" {
			updates["after"] = after
		}
		result := tx.Model(&model.PolicyRevision{}).Where("id = ?", id).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrNotFound
		}
		return tx.Create(&audit).Error
	})
}

func (r *peerRelayRepository) CompletePolicyUpdate(ctx context.Context, revisionID string, sourceNodeIDs []string, after string, assignments []model.RelayAssignment, audits []model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.PolicyRevision{}).Where("id = ? AND status = ?", revisionID, "pending").Updates(map[string]any{"after": after, "status": "success"})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrNotFound
		}
		if len(sourceNodeIDs) > 0 {
			if err := tx.Where("source_node_id IN ?", sourceNodeIDs).Delete(&model.RelayAssignment{}).Error; err != nil {
				return err
			}
		}
		if len(assignments) > 0 {
			if err := tx.Create(&assignments).Error; err != nil {
				return err
			}
		}
		if len(audits) > 0 {
			if err := tx.Create(&audits).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *peerRelayRepository) CompleteSelectorPolicyUpdate(ctx context.Context, revisionID, after string, rule *model.SelectorPolicyRule, deleteRuleID string, audits []model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.PolicyRevision{}).Where("id = ? AND status = ?", revisionID, "pending").Updates(map[string]any{"after": after, "status": "success"})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrNotFound
		}
		if rule != nil {
			if err := tx.Create(rule).Error; err != nil {
				return err
			}
		}
		if deleteRuleID != "" {
			result := tx.Delete(&model.SelectorPolicyRule{}, "id = ?", deleteRuleID)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrNotFound
			}
		}
		if len(audits) > 0 {
			if err := tx.Create(&audits).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *peerRelayRepository) DeleteSelectorPolicyRuleWithAudit(ctx context.Context, id string, audit model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Delete(&model.SelectorPolicyRule{}, "id = ?", id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrNotFound
		}
		return tx.Create(&audit).Error
	})
}

func (r *peerRelayRepository) Revisions(ctx context.Context, limit int) ([]model.PolicyRevision, error) {
	var values []model.PolicyRevision
	err := r.db.WithContext(ctx).Order("created_at DESC, id DESC").Limit(limit).Find(&values).Error
	return values, err
}

func (r *peerRelayRepository) Revision(ctx context.Context, id string) (model.PolicyRevision, error) {
	var value model.PolicyRevision
	err := r.db.WithContext(ctx).First(&value, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.PolicyRevision{}, ErrNotFound
	}
	return value, err
}

func (r *peerRelayRepository) LatestSuccessfulRevision(ctx context.Context) (model.PolicyRevision, error) {
	var value model.PolicyRevision
	err := r.db.WithContext(ctx).Where("status = ?", "success").Order("created_at DESC, id DESC").First(&value).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.PolicyRevision{}, nil
	}
	return value, err
}

func (r *peerRelayRepository) TrimRevisions(ctx context.Context, keep int) error {
	return r.db.WithContext(ctx).Exec(`DELETE FROM policy_revisions WHERE id NOT IN (SELECT id FROM policy_revisions WHERE status = ? ORDER BY created_at DESC, id DESC LIMIT ?)`, "success", keep).Error
}

func (r *peerRelayRepository) Audit(ctx context.Context, value model.AuditLog) error {
	return r.db.WithContext(ctx).Create(&value).Error
}

func (r *peerRelayRepository) Audits(ctx context.Context, limit int) ([]model.AuditLog, error) {
	var values []model.AuditLog
	err := r.db.WithContext(ctx).Order("created_at DESC, id DESC").Limit(limit).Find(&values).Error
	return values, err
}

func (r *peerRelayRepository) SaveSyncState(ctx context.Context, value model.SyncState) error {
	return r.db.WithContext(ctx).Save(&value).Error
}

func (r *peerRelayRepository) SyncState(ctx context.Context) (model.SyncState, error) {
	var value model.SyncState
	err := r.db.WithContext(ctx).First(&value, "key = ?", "headscale").Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SyncState{Key: "headscale"}, nil
	}
	return value, err
}

func (r *peerRelayRepository) PeerRelaySettings(ctx context.Context) (model.PeerRelaySettings, error) {
	var value model.PeerRelaySettings
	err := r.db.WithContext(ctx).First(&value, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.PeerRelaySettings{}, ErrNotFound
	}
	return value, err
}

func (r *peerRelayRepository) SavePeerRelaySettingsWithAudit(ctx context.Context, value model.PeerRelaySettings, audit model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&value).Error; err != nil {
			return err
		}
		return tx.Create(&audit).Error
	})
}
