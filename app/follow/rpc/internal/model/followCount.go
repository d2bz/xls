package model

import "gorm.io/gorm"

type FollowCount struct {
	UserID      uint64 `json:"user_id"`
	FollowCount int    `json:"follow_count"`
	FansCount   int    `json:"fans_count"`
}

func (FollowCount) TableName() string {
	return "follow_count"
}

func IncrFollowCount(db *gorm.DB, userID uint64) error {
	return db.Exec("INSERT INTO follow_count (user_id, follow_count) VALUES (?, 1) ON DUPLICATE KEY UPDATE follow_count = follow_count + 1", userID).Error
}

func DecrFollowCount(db *gorm.DB, userID uint64) error {
	return db.Exec("UPDATE follow_count SET follow_count = follow_count - 1 WHERE user_id = ? AND follow_count > 0", userID).Error
}

func IncrFansCount(db *gorm.DB, userID uint64) error {
	return db.Exec("INSERT INTO follow_count (user_id, fans_count) VALUES (?, 1) ON DUPLICATE KEY UPDATE follow_count = fans_count + 1", userID).Error
}

func DecrFansCount(db *gorm.DB, userID uint64) error {
	return db.Exec("UPDATE fans_count SET fans_count = fans_count - 1 WHERE user_id = ? AND fans_count > 0", userID).Error
}
