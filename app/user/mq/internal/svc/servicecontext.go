package svc

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
	"xls/app/user/mq/internal/config"
	"xls/app/user/mq/internal/model"
)

type ServiceContext struct {
	Config         config.Config
	KqPusherClient *kq.Pusher
	MysqlDB        *gorm.DB
	BizRedis       redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		MysqlDB:  model.InitMysql(c.Mysql.Datasource),
		BizRedis: *redis.MustNewRedis(c.BizRedis),
	}
}
