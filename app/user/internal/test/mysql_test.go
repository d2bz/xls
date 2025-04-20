package test

import (
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMysqlConn(t *testing.T) {
	db, err := gorm.Open(mysql.Open(
		"root:123456@tcp(localhost)/user?charset=utf8&parseTime=True&loc=Local",
	), &gorm.Config{})
	if err != nil {
		t.Fatalf("mysql connection failed: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal("failed to get sqlDB")
	}

	err = sqlDB.Ping()
	if err != nil {
		t.Fatal("mysql ping failed")
	} else {
		t.Log("mysql connection successful")
	}
}
