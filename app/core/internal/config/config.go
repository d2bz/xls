package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	BizRedis redis.RedisConf
	UserRPC  zrpc.RpcClientConf
	VideoRPC zrpc.RpcClientConf
	LikeRPC  zrpc.RpcClientConf
	Auth     AuthConf
	Minio    MinioConf
}

type AuthConf struct {
	AccessSecret string
	AccessExpire int64
}

type MinioConf struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	BaseUrl   string
}
