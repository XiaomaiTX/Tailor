package model

import "time"

type Example struct {
	ID          int64   `gorm:"primaryKey"`
	Name        string  `gorm:"size:120;not null"`
	Description *string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Example) TableName() string { return "examples" }
