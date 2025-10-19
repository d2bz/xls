package model

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitMysql(dataSource string) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dataSource), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("failed to connect database")
	}
	autoMigrate(db)
	return db
}

func autoMigrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&Follow{},
	)
	if err != nil {
		panic("failed to auto migrate database")
	}
}
