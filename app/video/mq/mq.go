package main

import (
	"flag"
	"fmt"

	"xls/app/video/mq/internal/config"
	"xls/app/video/mq/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
)

var configFile = flag.String("f", "etc/mq-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
}
