package svc

import (
	"xls/app/video/rpc/video/internal/config"
	"xls/app/video/rpc/video/model"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config  config.Config
	MysqlDB *gorm.DB
	BizRedis *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:  c,
		MysqlDB: model.InitMysql(c.Mysql.Datasource),
		BizRedis: redis.New(c.BizRedis.Host, redis.WithPass(c.BizRedis.Pass)),
	}
}
