package model

import "gorm.io/gorm"

type Video struct {
	Uid        uint   `gorm:"not null" json:"uid"`
	Title      string `gorm:"varchar(255);not null" json:"title"`
	Url        string `gorm:"varchar(255); not null" json:"url"`
	LikeNum    int    `json:"like_num"`
	CommentNum int    `json:"comment_num"`
	gorm.Model
}

func (v *Video) TableName() string {
	return "video"
}

func UpdateLikeCount(db *gorm.DB, videoID uint64, count int) error {
	return db.Model(&Video{}).Where("id = ?", videoID).Update("like_num", gorm.Expr("like_num + ?", count)).Error
}
