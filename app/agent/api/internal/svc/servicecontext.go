package svc

import (
	"xls/app/agent/api/internal/config"
	"xls/app/agent/rpc/agentclient"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config   config.Config
	AgentRpc agentclient.Agent
}

func NewServiceContext(c config.Config) *ServiceContext {
	cli, err := zrpc.NewClient(c.AgentRPC)
	if err != nil {
		panic("failed to create agent rpc client: " + err.Error())
	}
	return &ServiceContext{
		Config:   c,
		AgentRpc: agentclient.NewAgent(cli),
	}
}
