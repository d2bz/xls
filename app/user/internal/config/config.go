package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Mysql    MysqlConf
	BizRedis redis.RedisConf
	Auth     AuthConf
}

type MysqlConf struct {
	Datasource string
}

type AuthConf struct {
	AccessSecret string
	AccessExpire int64
}
