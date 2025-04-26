package main

import (
	"flag"
	"fmt"

	"xls/app/user/internal/config"
	"xls/app/user/internal/server"
	"xls/app/user/internal/svc"
	"xls/app/user/user"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/user-rpc.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		user.RegisterUserServer(grpcServer, server.NewUserServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Println("Loaded etcd hosts:", c.Etcd.Hosts)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
	// sudo bash -c 'echo -e "nameserver 8.8.8.8\nnameserver 114.114.114.114" > /etc/resolv.conf'
}
