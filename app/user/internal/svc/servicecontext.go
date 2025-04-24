package svc

import (
	"xls/app/user/internal/config"
	"xls/app/user/internal/model"

	"gorm.io/gorm"
)

type ServiceContext struct {
	Config  config.Config
	MysqlDB *gorm.DB
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:  c,
		MysqlDB: model.InitMysql(c.Mysql.Datasource),
	}
}
