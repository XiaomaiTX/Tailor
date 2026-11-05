package repository

import (
	"context"
	"errors"
	"time"

	"github.com/XiaomaiTX/go-react-starter/internal/model"
	"gorm.io/gorm"
)

var ErrAuthRecordNotFound = errors.New("authentication record not found")

type AuthRepository interface {
	SaveLoginState(context.Context, model.OIDCLoginState) error
	ConsumeLoginState(context.Context, string, time.Time) (model.OIDCLoginState, error)
	CreateAuthSession(context.Context, model.AuthSession) error
	AuthSession(context.Context, string, time.Time) (model.AuthSession, error)
	DeleteAuthSession(context.Context, string) error
	PurgeExpiredAuthRecords(context.Context, time.Time) error
}

type authRepository struct{ db *gorm.DB }

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) SaveLoginState(ctx context.Context, value model.OIDCLoginState) error {
	return r.db.WithContext(ctx).Create(&value).Error
}

func (r *authRepository) ConsumeLoginState(ctx context.Context, stateHash string, now time.Time) (model.OIDCLoginState, error) {
	var value model.OIDCLoginState
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&value, "state_hash = ? AND expires_at > ?", stateHash, now).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAuthRecordNotFound
			}
			return err
		}
		result := tx.Delete(&model.OIDCLoginState{}, "state_hash = ?", stateHash)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrAuthRecordNotFound
		}
		return nil
	})
	if err != nil {
		return model.OIDCLoginState{}, err
	}
	return value, nil
}

func (r *authRepository) CreateAuthSession(ctx context.Context, value model.AuthSession) error {
	return r.db.WithContext(ctx).Create(&value).Error
}

func (r *authRepository) AuthSession(ctx context.Context, tokenHash string, now time.Time) (model.AuthSession, error) {
	var value model.AuthSession
	err := r.db.WithContext(ctx).First(&value, "token_hash = ? AND expires_at > ?", tokenHash, now).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.AuthSession{}, ErrAuthRecordNotFound
	}
	return value, err
}

func (r *authRepository) DeleteAuthSession(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).Delete(&model.AuthSession{}, "token_hash = ?", tokenHash).Error
}

func (r *authRepository) PurgeExpiredAuthRecords(ctx context.Context, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.OIDCLoginState{}, "expires_at <= ?", now).Error; err != nil {
			return err
		}
		return tx.Delete(&model.AuthSession{}, "expires_at <= ?", now).Error
	})
}
