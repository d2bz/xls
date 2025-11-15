package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/gorm"
	"xls/app/user/rpc/userclient"
	"xls/app/video/mq/internal/config"
	"xls/app/video/mq/internal/model"
	"xls/pkg/es"
)

type ServiceContext struct {
	Config   config.Config
	UserRPC  userclient.User
	MysqlDB  *gorm.DB
	BizRedis redis.Redis
	Es       *es.Es
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		UserRPC:  userclient.NewUser(zrpc.MustNewClient(c.UserRPC)),
		MysqlDB:  model.InitMysql(c.Mysql.Datasource),
		BizRedis: *redis.MustNewRedis(c.BizRedis),
		Es: es.MustNewEs(&es.Config{
			Address:  c.Elasticsearch.Address,
			Username: c.Elasticsearch.Username,
			Password: c.Elasticsearch.Password,
		}),
	}
}
