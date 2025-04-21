package model

type User struct {
	ID       int64  `gorm:"primaryKey;autoIncremnt" json:"id"`
	Name     string `gorm:"type:varchar(100);not null" json:"name"`
	Email    string `gorm:"type:varchar(100);not null;unique" json:"email"`
	Password string `gorm:"type:varchar(100);not null" json:"password"`
	Avatar   string `gorm:"type:varchar(255);not null" json:"avatar"`
}

func (*User) TableName() string {
	return "user"
}
