package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type Config struct {
	KqConsumerConf kq.KqConf
	Mysql struct {
		Datasource string
	}
	BizRedis redis.RedisConf
}
