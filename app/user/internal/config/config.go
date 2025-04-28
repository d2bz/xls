package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	Mysql Mysql
	Auth  Auth
}

type Mysql struct {
	Datasource string
}

type Auth struct {
	AccessSecret string
	AccessExpire int64
}
