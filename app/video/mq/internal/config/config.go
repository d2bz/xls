package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Config struct {
	LikeSyncKqConsumerConf kq.KqConf
	VideoKqConsumerConf    kq.KqConf
	Mysql                  struct {
		Datasource string
	}
	BizRedis      redis.RedisConf
	Elasticsearch struct {
		Address  []string
		Username string
		Password string
	}
}
