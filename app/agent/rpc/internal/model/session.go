package model

import (
	"time"

	"gorm.io/gorm"
)

type Session struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UUID      string `gorm:"type:varchar(64);uniqueIndex;not null" json:"uuid"`
	UserID    uint64 `gorm:"index;not null" json:"user_id"`
	Title     string `gorm:"type:varchar(255);not null;default:'新会话'" json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Session) TableName() string {
	return "session"
}

type SessionMessage struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	SessionID uint   `gorm:"index;not null" json:"session_id"`
	Role      string `gorm:"type:varchar(32);not null" json:"role"`
	Content   string `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (SessionMessage) TableName() string {
	return "session_message"
}
