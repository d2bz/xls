package model

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"time"
)

type Follow struct {
	ID             uint64         `gorm:"primary_key" json:"id"`
	UserID         uint64         `json:"user_id"`
	FollowedUserID uint64         `json:"followed_user_id"`
	FollowStatus   int            `json:"follow_status"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

type FollowModel struct {
	db *gorm.DB
}

func NewFollowModel(db *gorm.DB) *FollowModel {
	return &FollowModel{
		db: db,
	}
}

func (*Follow) TableName() string {
	return "follow"
}

func FollowInsert(db *gorm.DB, data *Follow) error {
	return db.Create(data).Error
}

func FollowFindByUserIDAndFollowedUserID(db *gorm.DB, userID uint64, followedUserID uint64) (*Follow, error) {
	var f *Follow
	err := db.Where("user_id = ? AND followed_user_id = ?", userID, followedUserID).First(f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return f, err
}

func FollowUpdateFields(db *gorm.DB, id uint64, values map[string]interface{}) error {
	return db.Model(&Follow{}).Where("id = ?", id).Updates(values).Error
}

func (m *FollowModel) FindByFollowedUserIDs(ctx context.Context, userID uint64, followedUserIDs []uint64) ([]*Follow, error) {
	var results []*Follow
	err := m.db.WithContext(ctx).
		Where("user_id = ? AND followed_user_id IN (?)", userID, followedUserIDs).
		Order("created_at DESC").
		Find(&results).Error

	return results, err
}

func (m *FollowModel) FindByUserID(ctx context.Context, userID uint64, limit int) ([]*Follow, error) {
	var result []*Follow
	err := m.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&result).Error

	return result, err
}
