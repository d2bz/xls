package model

import "gorm.io/gorm"

type Like struct {
	UserID     uint64 `gorm:"not null" json:"user_id"`
	TargetID   uint64 `gorm:"not null" json:"target_id"`
	TargetType int32  `gorm:"not null" json:"target_type"`
	gorm.Model
}

func (*Like) TableName() string {
	return "like"
}
