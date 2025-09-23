package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	Mysql    MysqlConf
	BizRedis redis.RedisConf
}

type MysqlConf struct {
	Datasource string
}
