package model

import "gorm.io/gorm"

type Tag struct {
	Name string `gorm:"unique;not null" json:"name"`

	gorm.Model
}

func (t *Tag) TableName() string {
	return "tag"
}
