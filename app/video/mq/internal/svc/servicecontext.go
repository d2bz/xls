package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
	"xls/app/video/mq/internal/config"
	"xls/app/video/rpc/video/model"
)

type ServiceContext struct {
	Config         config.Config
	MysqlDB        *gorm.DB
	BizRedis       redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:         c,
		MysqlDB:        model.InitMysql(c.Mysql.Datasource),
		BizRedis:       *redis.MustNewRedis(c.BizRedis),
	}
}
