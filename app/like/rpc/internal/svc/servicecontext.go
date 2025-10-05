package svc

import (
	"xls/app/like/rpc/internal/config"
	"xls/app/like/rpc/internal/model"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config         config.Config
	MysqlDB        *gorm.DB
	BizRedis       redis.Redis
	KqPusherClient *kq.Pusher
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:         c,
		MysqlDB:        model.InitMysql(c.Mysql.Datasource),
		BizRedis:       *redis.MustNewRedis(c.BizRedis),
		KqPusherClient: kq.NewPusher(c.KqPusherConf.Brokers, c.KqPusherConf.Topic),
	}
}
