package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	Mysql Mysql
}

type Mysql struct {
	Datasource string
}
