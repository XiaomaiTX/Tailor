package model

import "time"

type Example struct {
	ID          int64     `gorm:"primaryKey;autoIncrement;index:idx_examples_created_at_id,sort:desc,priority:2"`
	Name        string    `gorm:"size:120;not null"`
	Description *string   `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_examples_created_at_id,sort:desc,priority:1"`
	UpdatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (Example) TableName() string { return "examples" }
