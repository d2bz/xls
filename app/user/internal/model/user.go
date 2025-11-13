package model

import (
	"encoding/json"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name     string `gorm:"type:varchar(100);not null" json:"name"`
	Email    string `gorm:"type:varchar(100);not null;unique" json:"email"`
	Password string `gorm:"type:varchar(100);not null" json:"password"`
	Avatar   string `gorm:"type:varchar(255);" json:"avatar"`
}

type UserModel struct {
	db *gorm.DB
}

func NewUserModel(db *gorm.DB) *UserModel {
	return &UserModel{
		db: db,
	}
}

func (*User) TableName() string {
	return "user"
}

func (u *User) ToString() (string, error) {
	userStr, err := json.Marshal(u)
	return string(userStr), err
}

func (u *User) FromString(userStr string) error {
	return json.Unmarshal([]byte(userStr), u)
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

func (um *UserModel) FindUserByID(id uint64) (*User, error) {
	var user *User
	if err := um.db.Where("id = ?", id).First(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}
