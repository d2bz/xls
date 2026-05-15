package svc

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/gorm"
	"xls/app/user/rpc/userclient"
	"xls/app/video/mq/internal/config"
	"xls/app/video/mq/internal/embedding"
	"xls/app/video/mq/internal/milvus"
	"xls/app/video/mq/internal/model"
	"xls/pkg/es"
)

type ServiceContext struct {
	Config   config.Config
	UserRPC  userclient.User
	MysqlDB  *gorm.DB
	BizRedis redis.Redis
	Es       *es.Es
	Milvus   *milvus.Client
	Embedder *embedding.Embedder
}

func NewServiceContext(c config.Config) *ServiceContext {
	ctx := context.Background()

	// Milvus 客户端
	milvusClient, err := milvus.NewClient(ctx, c.Milvus.Address, c.Milvus.Collection)
	if err != nil {
		panic("failed to connect milvus: " + err.Error())
	}

	// Embedding 客户端
	embedder, err := embedding.NewEmbedder(
		ctx,
		c.Ark.APIKey,
		c.Ark.Model,
		c.Ark.BaseURL,
		c.Ark.Region,
		c.Embedding.Timeout,
	)
	if err != nil {
		panic("failed to create embedder: " + err.Error())
	}

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
		Milvus:   milvusClient,
		Embedder: embedder,
	}
}
