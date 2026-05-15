package model

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitMysql(dataSource string, maxIdle, maxOpen int) *gorm.DB {
	db, err := gorm.Open(mysql.Open(dataSource), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get underlying sql.DB: " + err.Error())
	}
	if maxIdle > 0 {
		sqlDB.SetMaxIdleConns(maxIdle)
	}
	if maxOpen > 0 {
		sqlDB.SetMaxOpenConns(maxOpen)
	} else {
		sqlDB.SetMaxOpenConns(100)
	}

	autoMigrate(db)
	return db
}

func InitMysqlSimple(dataSource string) *gorm.DB {
	return InitMysql(dataSource, 10, 100)
}

func autoMigrate(db *gorm.DB) {
	err := db.AutoMigrate(&Session{}, &SessionMessage{})
	if err != nil {
		panic("failed to auto migrate session tables: " + err.Error())
	}
}
