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

func (v *Video) Insert(db *gorm.DB) error {
	return db.Create(v).Error
}

func FindVideoByID(db *gorm.DB, videoID uint) (*Video, error) {
	var video *Video
	err := db.Where("id = ?", videoID).First(video).Error
	return video, err
}
