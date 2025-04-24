package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	ID       int64  `gorm:"primaryKey;autoIncremnt" json:"id"`
	Name     string `gorm:"type:varchar(100);not null" json:"name"`
	Email    string `gorm:"type:varchar(100);not null;unique" json:"email"`
	Password string `gorm:"type:varchar(100);not null" json:"password"`
	Avatar   string `gorm:"type:varchar(255);not null" json:"avatar"`
}

func (*User) TableName() string {
	return "user"
}

func GetUserByEmail(db *gorm.DB, email string) (*User, error) {
	var user User
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *User) Insert(db *gorm.DB) error {
	err := db.Create(u).Error
	return err
}
