package svc

import (
	"xls/app/core/internal/config"
	"xls/app/user/userclient"
	"xls/app/video/rpc/video/videoclient"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config   config.Config
	BizRedis *redis.Redis
	UserRpc  userclient.User
	VideoRpc videoclient.Video
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		BizRedis: redis.New(c.BizRedis.Host, redis.WithPass(c.BizRedis.Pass)),
		UserRpc:  userclient.NewUser(zrpc.MustNewClient(c.UserRPC)),
		VideoRpc: videoclient.NewVideo(zrpc.MustNewClient(c.VideoRPC)),
	}
}
