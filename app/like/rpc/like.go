package main

import (
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/threading"
	"xls/app/like/rpc/internal/cron"

	"xls/app/like/rpc/internal/config"
	"xls/app/like/rpc/internal/server"
	"xls/app/like/rpc/internal/svc"
	"xls/app/like/rpc/like"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/like-rpc.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		like.RegisterLikeServer(grpcServer, server.NewLikeServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	threading.GoSafe(func() {
		cron.ScheduledTask(ctx)
	})

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
