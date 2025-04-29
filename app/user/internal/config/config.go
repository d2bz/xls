package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Mysql    Mysql
	BizRedis redis.RedisConf
	Auth     Auth
}

type Mysql struct {
	Datasource string
}

type Auth struct {
	AccessSecret string
	AccessExpire int64
}
