package svc

import (
	"xls/app/comment/rpc/commentclient"
	"xls/app/core/internal/config"
	"xls/app/like/rpc/likeclient"
	"xls/app/user/userclient"
	"xls/app/video/rpc/video/videoclient"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config     config.Config
	BizRedis   *redis.Redis
	UserRpc    userclient.User
	VideoRpc   videoclient.Video
	LikeRpc    likeclient.Like
	CommentRpc commentclient.Comment
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:     c,
		BizRedis:   redis.MustNewRedis(c.BizRedis),
		UserRpc:    userclient.NewUser(zrpc.MustNewClient(c.UserRPC)),
		VideoRpc:   videoclient.NewVideo(zrpc.MustNewClient(c.VideoRPC)),
		LikeRpc:    likeclient.NewLike(zrpc.MustNewClient(c.LikeRPC)),
		CommentRpc: commentclient.NewComment(zrpc.MustNewClient(c.CommentRPC)),
	}
}
