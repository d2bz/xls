package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	LikeSyncKqConsumerConf kq.KqConf
	VideoKqConsumerConf    kq.KqConf
	TagKqConsumerConf      kq.KqConf
	UserRPC                zrpc.RpcClientConf
	Mysql                  struct {
		Datasource string
	}
	BizRedis      redis.RedisConf
	Elasticsearch struct {
		Address  []string
		Username string
		Password string
	}
	Milvus struct {
		Address      string
		Collection   string
		VectorDim    int64
	}
	Ark       ArkConf
	Embedding EmbeddingConf
}

type ArkConf struct {
	APIKey  string
	Model   string
	BaseURL string
	Region  string
}

type EmbeddingConf struct {
	Timeout int
}
