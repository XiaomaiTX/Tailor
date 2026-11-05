package model

import "time"

type OIDCLoginState struct {
	StateHash    string    `gorm:"primaryKey;size:64"`
	Nonce        string    `gorm:"size:128;not null"`
	CodeVerifier string    `gorm:"size:128;not null"`
	CreatedAt    time.Time `gorm:"not null"`
	ExpiresAt    time.Time `gorm:"not null;index"`
}

func (OIDCLoginState) TableName() string { return "oidc_login_states" }

type AuthSession struct {
	TokenHash       string    `gorm:"primaryKey;size:64"`
	HeadscaleUserID string    `gorm:"size:120;not null;index"`
	CreatedAt       time.Time `gorm:"not null"`
	ExpiresAt       time.Time `gorm:"not null;index"`
}

func (AuthSession) TableName() string { return "auth_sessions" }
