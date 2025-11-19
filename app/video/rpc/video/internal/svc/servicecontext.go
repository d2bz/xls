package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/gorm"
	"xls/app/like/rpc/likeclient"
	"xls/app/video/rpc/video/internal/config"
	"xls/app/video/rpc/video/internal/model"
	"xls/pkg/es"
)

type ServiceContext struct {
	Config   config.Config
	LikeRPC  likeclient.Like
	MysqlDB  *gorm.DB
	BizRedis *redis.Redis
	TypedEs  *es.TypedEs
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		LikeRPC:  likeclient.NewLike(zrpc.MustNewClient(c.LikeRPC)),
		MysqlDB:  model.InitMysql(c.Mysql.Datasource),
		BizRedis: redis.MustNewRedis(c.BizRedis),
		TypedEs: es.MustNewTypedEs(&es.Config{
			Address:  c.Elasticsearch.Address,
			Username: c.Elasticsearch.Username,
			Password: c.Elasticsearch.Password,
		}),
	}
}
