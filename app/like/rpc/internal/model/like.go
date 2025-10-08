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

func (like *Like) IsLike(db *gorm.DB, targetID, uid uint64) (bool, error) {
	err := db.Where("target_id = ? AND user_id = ?", targetID, uid).First(like).Error
	if err != nil && like.ID == 0 {
		return false, err
	}
	return true, nil
}
