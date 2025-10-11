package model

import "gorm.io/gorm"

type Comment struct {
	gorm.Model
	UserID       uint64 `gorm:"not null" json:"user_id"`
	TargetID     uint64 `gorm:"not null" json:"target_id"`
	TargetUserID uint64 `gorm:"not null" json:"target_user_id"`
	ParentID     uint64 `gorm:"not null" json:"parent_id"`
	Content      string `gorm:"not null" json:"content"`
	LikeNum      uint64 `gorm:"not null;default:0" json:"like_num"`
}

func (*Comment) TableName() string {
	return "comment"
}

func (c *Comment) InsertComment(db *gorm.DB) error {
	return db.Create(c).Error
}
