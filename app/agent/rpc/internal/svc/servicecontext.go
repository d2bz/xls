package svc

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"xls/app/agent/rpc/internal/agentcore"
	"xls/app/agent/rpc/internal/agentcore/memory"
	"xls/app/agent/rpc/internal/agentcore/milvus"
	"xls/app/agent/rpc/internal/config"
	"xls/app/agent/rpc/internal/mcpclient"
	"xls/app/agent/rpc/internal/model"

	"github.com/zeromicro/go-zero/zrpc"
	"xls/app/follow/rpc/followclient"
	"xls/app/like/rpc/likeclient"
	"xls/app/user/rpc/userclient"
	"xls/app/video/rpc/video/videoclient"
	"xls/pkg/embedding"
)

type ServiceContext struct {
	Config    config.Config
	VideoRpc  videoclient.Video
	UserRpc   userclient.User
	FollowRpc followclient.Follow
	LikeRpc   likeclient.Like

	// AgentCore 包含 adk.Runner，支持多轮对话。
	// 通过 AgentCore.Runner.Run(ctx, history) 调用。
	AgentCore *agentcore.AgentCore
	// SessionStore 负责会话消息的持久化。
	SessionStore memory.SessionStore
	// VideoMCPClient 连接 aigroup-video-mcp MCP 服务器。
	VideoMCPClient *mcpclient.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	userCli := zrpc.MustNewClient(c.UserRPC)
	videoCli := zrpc.MustNewClient(c.VideoRPC)
	followCli := zrpc.MustNewClient(c.FollowRPC)
	likeCli := zrpc.MustNewClient(c.LikeRPC)

	videoRpc := videoclient.NewVideo(videoCli)
	userRpc := userclient.NewUser(userCli)
	followRpc := followclient.NewFollow(followCli)
	likeRpc := likeclient.NewLike(likeCli)

	ctx := context.Background()

	// 创建 Embedder
	var embedder *embedding.Embedder
	{
		e, err := embedding.NewEmbedder(ctx,
			c.Agent.Model.Ark.APIKey,
			c.Agent.Model.Ark.Model,
			c.Agent.Model.Ark.BaseURL,
			c.Agent.Model.Ark.Region,
			30,
		)
		if err != nil {
			panic("create embedder: " + err.Error())
		}
		embedder = e
	}

	// 创建 Milvus Client
	var milvusClient *milvus.Client
	{
		mc, err := milvus.NewClient(ctx,
			c.Milvus.Address,
			c.Milvus.Collection,
		)
		if err != nil {
			panic("create milvus client: " + err.Error())
		}
		milvusClient = mc
	}

	// 创建 Session Store
	sessionStore := createSessionStore(c)

	// 创建 MCP 客户端（连接 aigroup-video-mcp 服务器）
	mcpClient := mcpclient.MustNewClient(ctx, c.Agent.VideoMCP)

	// 收集 MCP 工具
	var mcpTools []tool.BaseTool
	if mcpClient != nil {
		mcpTools = mcpClient.Tools()
	}

	deps := agentcore.AgentDeps{
		VideoRpc:  videoRpc,
		UserRpc:   userRpc,
		FollowRpc: followRpc,
		LikeRpc:   likeRpc,
		Embedder:  embedder,
		Milvus:    milvusClient,
		MCPTools:  mcpTools,
	}

	agentCfg := agentcore.AgentConfig{
		ArkAPIKey:  c.Agent.Model.Ark.APIKey,
		ArkModel:   c.Agent.Model.Ark.Model,
		ArkBaseURL: c.Agent.Model.Ark.BaseURL,
		ArkRegion:  c.Agent.Model.Ark.Region,
		MaxIter:    c.Agent.MaxIter,
		MaxSteps:   6,
		MilvusCfg: agentcore.MilvusClientConfig{
			Address:      c.Milvus.Address,
			Collection:   c.Milvus.Collection,
			SearchTopK:   c.Milvus.SearchTopK,
			SearchNProbe: c.Milvus.SearchNProbe,
		},
		MiddlewareCfg: agentcore.MiddlewareConfig{
			Summarization: agentcore.SummarizationConfig{
				Enabled:         c.Agent.Middleware.Summarization.Enabled,
				TriggerTokens:   c.Agent.Middleware.Summarization.TriggerTokens,
				TriggerMessages: c.Agent.Middleware.Summarization.TriggerMessages,
				PreserveTokens:  c.Agent.Middleware.Summarization.PreserveTokens,
				TranscriptPath:  c.Agent.Middleware.Summarization.TranscriptPath,
				Instruction:     c.Agent.Middleware.Summarization.Instruction,
			},
			Reduction: agentcore.ReductionConfig{
				Enabled:        c.Agent.Middleware.Reduction.Enabled,
				MaxLengthTrunc: c.Agent.Middleware.Reduction.MaxLengthTrunc,
				MaxTokensClear: c.Agent.Middleware.Reduction.MaxTokensClear,
				RootDir:        c.Agent.Middleware.Reduction.RootDir,
			},
		},
	}

	agentCore, err := agentcore.MustInit(ctx, deps, agentCfg)
	if err != nil {
		panic("create agent core: " + err.Error())
	}

	return &ServiceContext{
		Config:          c,
		VideoRpc:        videoRpc,
		UserRpc:         userRpc,
		FollowRpc:       followRpc,
		LikeRpc:         likeRpc,
		AgentCore:       agentCore,
		SessionStore:    sessionStore,
		VideoMCPClient:  mcpClient,
	}
}

func createSessionStore(c config.Config) memory.SessionStore {
	switch c.Agent.Session.Type {
	case "mysql":
		db := model.InitMysql(c.Agent.Session.MySQL.DSN, c.Agent.Session.MySQL.MaxIdleConns, c.Agent.Session.MySQL.MaxOpenConns)
		return memory.NewSessionStoreImpl(db, c.Agent.Session.MaxHistory)
	case "memory":
		return memory.NewInMemorySessionStoreAdapter()
	default:
		db := model.InitMysql(c.Agent.Session.MySQL.DSN, c.Agent.Session.MySQL.MaxIdleConns, c.Agent.Session.MySQL.MaxOpenConns)
		return memory.NewSessionStoreImpl(db, c.Agent.Session.MaxHistory)
	}
}
