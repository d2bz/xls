package agentcore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/reduction"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	agentpb "xls/app/agent/rpc/agent"
	"xls/app/agent/rpc/internal/agentcore/milvus"
	"xls/app/agent/rpc/internal/agentcore/router"
	"xls/app/agent/rpc/internal/agentcore/tools"
	"xls/app/follow/rpc/followclient"
	"xls/app/like/rpc/likeclient"
	"xls/app/user/rpc/userclient"
	"xls/app/video/rpc/video/videoclient"
	"xls/pkg/embedding"
)

// AgentDeps holds all RPC clients and external service dependencies.
type AgentDeps struct {
	VideoRpc  videoclient.Video
	UserRpc   userclient.User
	FollowRpc followclient.Follow
	LikeRpc   likeclient.Like
	Embedder  *embedding.Embedder
	Milvus    *milvus.Client
	MCPTools  []tool.BaseTool // video analysis tools from aigroup-video-mcp MCP server
}

type AgentConfig struct {
	ArkAPIKey     string
	ArkModel      string
	ArkBaseURL    string
	ArkRegion     string
	MaxIter       int
	MaxSteps      int
	MilvusCfg     MilvusClientConfig
	MiddlewareCfg MiddlewareConfig
}

type MilvusClientConfig struct {
	Address      string
	Collection   string
	SearchTopK   int
	SearchNProbe int
}

// MiddlewareConfig holds configuration for ADK ChatModelAgentMiddleware.
type MiddlewareConfig struct {
	Summarization SummarizationConfig
	Reduction     ReductionConfig
}

// SummarizationConfig configures the summarization middleware.
type SummarizationConfig struct {
	Enabled         bool
	TriggerTokens   int
	TriggerMessages int
	PreserveTokens  int
	TranscriptPath  string
	Instruction     string
}

// ReductionConfig configures the tool reduction middleware.
type ReductionConfig struct {
	Enabled        bool
	MaxLengthTrunc int
	MaxTokensClear int64
	RootDir        string
}

// AgentCore 封装 adk.Runner 和 ChatModelAgent，支持多轮对话。
type AgentCore struct {
	Runner *adk.Runner
}

// MustInit 初始化 Agent，返回 AgentCore（包含 adk.Runner）。
// Runner 自动管理多轮对话历史，SessionStore 负责持久化。
func MustInit(ctx context.Context, deps AgentDeps, cfg AgentConfig) (*AgentCore, error) {
	// 1. 创建 ChatModel（实现 ToolCallingChatModel 接口）
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:  cfg.ArkAPIKey,
		Model:   cfg.ArkModel,
		BaseURL: cfg.ArkBaseURL,
		Region:  cfg.ArkRegion,
	})
	if err != nil {
		return nil, fmt.Errorf("create chat model failed: %w", err)
	}

	// 2. 创建工具集（业务工具 + MCP 视频分析工具）
	allTools, err := tools.NewTools(deps.VideoRpc, deps.UserRpc, deps.FollowRpc, deps.LikeRpc, deps.Embedder, deps.Milvus)
	if err != nil {
		return nil, fmt.Errorf("create tools failed: %w", err)
	}
	// 追加 MCP 视频分析工具
	if len(deps.MCPTools) > 0 {
		allTools = append(allTools, deps.MCPTools...)
	}

	// 3. 提取 MCP 工具的 ToolInfo（用于动态注入到 prompt）
	mcpToolsInfo := make([]*schema.ToolInfo, 0, len(deps.MCPTools))
	for _, t := range deps.MCPTools {
		info, err := t.Info(ctx)
		if err != nil {
			return nil, fmt.Errorf("get MCP tool info failed: %w", err)
		}
		mcpToolsInfo = append(mcpToolsInfo, info)
	}

	// 4. 将业务工具绑定到模型
	toolsInfo := make([]*schema.ToolInfo, 0, len(allTools))
	for _, t := range allTools {
		info, err := t.Info(ctx)
		if err != nil {
			return nil, fmt.Errorf("get tool info failed: %w", err)
		}
		toolsInfo = append(toolsInfo, info)
	}

	toolChatModel, err := chatModel.WithTools(toolsInfo)
	if err != nil {
		return nil, fmt.Errorf("bind tools failed: %w", err)
	}

	// 5. 创建意图路由 Graph（现有逻辑）
	intentGraph, err := router.NewIntentRouterGraph(ctx, router.RouterGraphConfig{
		ChatModel:        chatModel,
		ToolChatModel:    toolChatModel,
		Tools:            allTools,
		MCPToolsInfo:     mcpToolsInfo,
		SupervisorPrompt: router.BuildSupervisorPromptWithoutMCP(),
		MaxSteps:         cfg.MaxSteps,
		MaxLoopIter:      cfg.MaxIter,
		Embedder:         deps.Embedder,
		Milvus:           deps.Milvus,
		MilvusSearchTopK: cfg.MilvusCfg.SearchTopK,
	})
	if err != nil {
		return nil, fmt.Errorf("create intent router graph failed: %w", err)
	}

	// 5. 将 Graph 包装为 tool，供 ChatModelAgent 调用
	graphTool, err := newRouterGraphTool(intentGraph)
	if err != nil {
		return nil, fmt.Errorf("wrap graph as tool failed: %w", err)
	}

	// 6. 构建 ChatModelAgentMiddleware
	handlers, err := buildMiddlewareHandlers(ctx, chatModel, cfg.MiddlewareCfg)
	if err != nil {
		return nil, fmt.Errorf("build middleware handlers: %w", err)
	}

	// 7. 创建 ChatModelAgent
	agent, err := newChatModelAgent(ctx, chatModel, toolChatModel, graphTool, handlers)
	if err != nil {
		return nil, fmt.Errorf("create chat model agent failed: %w", err)
	}

	// 8. 创建 Runner
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: false,
	})

	return &AgentCore{Runner: runner}, nil
}

func newChatModelAgent(
	ctx context.Context,
	chatModel model.BaseChatModel,
	toolChatModel model.ToolCallingChatModel,
	routerTool tool.BaseTool,
	handlers []adk.ChatModelAgentMiddleware,
) (*adk.ChatModelAgent, error) {
	instruction := `你是一个友好的短视频平台AI助手。
你可以通过调用工具来帮助用户完成各种任务，包括：
- 搜索视频（video_router_tool）
- 获取视频推荐
- 分析视频数据
- 查询用户关系（关注/粉丝）

重要规则：
1. 当用户询问视频相关内容时，优先调用 video_router_tool。
2. 如果用户只是闲聊或打招呼，直接回答，不需要调用工具。
3. 如果你不确定用户的意图，直接调用 video_router_tool，让它来判断。
4. 回复应该简洁友好，符合中文口语习惯。`

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "VideoAssistant",
		Description: "短视频平台AI助手。当需要搜索视频、分析视频、推荐视频、查询粉丝关系时，必须调用此工具。",
		Instruction: instruction,
		Model:       toolChatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{routerTool},
			},
			ReturnDirectly: map[string]bool{
				"video_router_tool": true,
			},
		},
		Handlers:      handlers,
		MaxIterations: 10,
	})
}

// routerGraphTool 包装 compose.Graph[string,string] 为 eino tool.BaseTool。
type routerGraphTool struct {
	name        string
	description string
	graph       compose.Runnable[string, string]
}

func newRouterGraphTool(graph compose.Runnable[string, string]) (tool.BaseTool, error) {
	t := &routerGraphTool{
		name:        "video_router_tool",
		description: "短视频平台路由工具。根据用户查询判断意图（搜索/推荐/分析/关系），调度对应工作流，返回结构化结果。",
		graph:       graph,
	}
	return t, nil
}

func (t *routerGraphTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: t.name,
		Desc: t.description,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     schema.String,
				Desc:     "用户的原始查询内容",
				Required: true,
			},
			"user_id": {
				Type:     schema.Integer,
				Desc:     "当前用户ID",
				Required: false,
			},
		}),
	}, nil
}

func (t *routerGraphTool) InvokableRun(ctx context.Context, params string, opts ...tool.Option) (string, error) {
	var p struct {
		Query  string `json:"query"`
		UserID uint64 `json:"user_id,omitempty"`
	}
	if err := json.Unmarshal([]byte(params), &p); err != nil {
		return "", fmt.Errorf("invalid params: %w", err)
	}

	result, err := t.graph.Invoke(ctx, p.Query)
	if err != nil {
		return "", fmt.Errorf("graph invoke failed: %w", err)
	}
	return result, nil
}

// MustInitLegacy 返回纯 compose.Runnable[string,string]，兼容旧调用方式（不走多轮）。
func MustInitLegacy(ctx context.Context, deps AgentDeps, cfg AgentConfig) (compose.Runnable[string, string], error) {
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:  cfg.ArkAPIKey,
		Model:   cfg.ArkModel,
		BaseURL: cfg.ArkBaseURL,
		Region:  cfg.ArkRegion,
	})
	if err != nil {
		return nil, err
	}

	allTools, err := tools.NewTools(deps.VideoRpc, deps.UserRpc, deps.FollowRpc, deps.LikeRpc, deps.Embedder, deps.Milvus)
	if err != nil {
		return nil, err
	}

	toolsInfo := make([]*schema.ToolInfo, 0, len(allTools))
	for _, t := range allTools {
		info, err := t.Info(ctx)
		if err != nil {
			return nil, err
		}
		toolsInfo = append(toolsInfo, info)
	}

	toolChatModel, err := chatModel.WithTools(toolsInfo)
	if err != nil {
		return nil, err
	}

	return router.NewIntentRouterGraph(ctx, router.RouterGraphConfig{
		ChatModel:        chatModel,
		ToolChatModel:    toolChatModel,
		Tools:            allTools,
		SupervisorPrompt: router.BuildSupervisorPromptWithoutMCP(),
		MaxSteps:         cfg.MaxSteps,
		MaxLoopIter:      cfg.MaxIter,
		Embedder:         deps.Embedder,
		Milvus:           deps.Milvus,
		MilvusSearchTopK: cfg.MilvusCfg.SearchTopK,
		MCPToolsInfo:     nil,
	})
}

// ExtractTextFromEvents 从 Runner 返回的 AgentEvent 流中提取文本响应。
func ExtractTextFromEvents(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder
	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return sb.String(), event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput
		if mv.Role != schema.Assistant {
			continue
		}
		if mv.Message != nil && mv.Message.Content != "" {
			sb.WriteString(mv.Message.Content)
		}
	}
	return sb.String(), nil
}

// buildMiddlewareHandlers 根据配置构建 ChatModelAgentMiddleware 列表。
// middleware 按注册顺序执行，summarization 需要在 reduction 之前执行，
// 因为 summarization 压缩的是整个对话历史，reduction 处理的是工具输出。
func buildMiddlewareHandlers(ctx context.Context, chatModel model.BaseChatModel, cfg MiddlewareConfig) ([]adk.ChatModelAgentMiddleware, error) {
	var handlers []adk.ChatModelAgentMiddleware

	if cfg.Summarization.Enabled {
		summaryCfg := &summarization.Config{
			Model: chatModel,
		}

		if cfg.Summarization.TriggerTokens > 0 {
			trigger := &summarization.TriggerCondition{
				ContextTokens: cfg.Summarization.TriggerTokens,
			}
			if cfg.Summarization.TriggerMessages > 0 {
				trigger.ContextMessages = cfg.Summarization.TriggerMessages
			}
			summaryCfg.Trigger = trigger
		}

		if cfg.Summarization.PreserveTokens > 0 {
			summaryCfg.PreserveUserMessages = &summarization.PreserveUserMessages{
				Enabled:   true,
				MaxTokens: cfg.Summarization.PreserveTokens,
			}
		} else {
			// 默认启用，保留最近 1/3 token 的用户消息
			summaryCfg.PreserveUserMessages = &summarization.PreserveUserMessages{
				Enabled: true,
			}
		}

		if cfg.Summarization.TranscriptPath != "" {
			summaryCfg.TranscriptFilePath = cfg.Summarization.TranscriptPath
		}

		if cfg.Summarization.Instruction != "" {
			summaryCfg.UserInstruction = cfg.Summarization.Instruction
		}

		summaryMW, err := summarization.New(ctx, summaryCfg)
		if err != nil {
			return nil, fmt.Errorf("create summarization middleware: %w", err)
		}
		handlers = append(handlers, summaryMW)
	}

	if cfg.Reduction.Enabled {
		reduceCfg := &reduction.Config{
			Backend: filesystem.NewInMemoryBackend(),
		}

		if cfg.Reduction.MaxLengthTrunc > 0 {
			reduceCfg.MaxLengthForTrunc = cfg.Reduction.MaxLengthTrunc
		}
		if cfg.Reduction.MaxTokensClear > 0 {
			reduceCfg.MaxTokensForClear = cfg.Reduction.MaxTokensClear
		}
		if cfg.Reduction.RootDir != "" {
			reduceCfg.RootDir = cfg.Reduction.RootDir
		}

		reduceMW, err := reduction.New(ctx, reduceCfg)
		if err != nil {
			return nil, fmt.Errorf("create reduction middleware: %w", err)
		}
		handlers = append(handlers, reduceMW)
	}

	return handlers, nil
}

// BuildQueryWithParams 将请求参数编码为带前缀的 query 字符串。
// Supervisor 会解析前缀并存入 State，后续节点通过 ProcessState 读取。
func BuildQueryWithParams(query string, videoID int64, keyword string, page, pageSize int, userID uint64) string {
	if videoID == 0 && keyword == "" && page == 0 && pageSize == 0 && userID == 0 {
		return query
	}
	var parts []string
	parts = append(parts, "## 请求参数")
	if videoID > 0 {
		parts = append(parts, "video_id="+int64ToString(videoID))
	}
	if keyword != "" {
		parts = append(parts, "keyword="+keyword)
	}
	if page > 0 {
		parts = append(parts, "page="+int64ToString(int64(page)))
	}
	if pageSize > 0 {
		parts = append(parts, "page_size="+int64ToString(int64(pageSize)))
	}
	if userID > 0 {
		parts = append(parts, "user_id="+uint64ToString(userID))
	}
	parts = append(parts, "", "## 用户查询", query)
	return strings.Join(parts, "\n")
}

func int64ToString(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func uint64ToString(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// FillStructuredResponse 从 Session 中提取工作流执行结果，填充到 ChatResponse 的结构化字段。
// 当前实现：解析 Session 最后一条 assistant 消息中的 JSON 数据，提取 videos 和 total。
func FillStructuredResponse(sessionUUID string, store *SessionStore, resp *agentpb.ChatResponse) {
	if store == nil {
		return
	}
	msgs, err := store.GetMessages(context.Background(), sessionUUID)
	if err != nil || len(msgs) == 0 {
		return
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == schema.Assistant && msgs[i].Content != "" {
			extractVideosFromAnswer(msgs[i].Content, resp)
			return
		}
	}
}

// extractVideosFromAnswer 解析 assistant 消息中的 JSON 数据，填充到 resp。
func extractVideosFromAnswer(content string, resp *agentpb.ChatResponse) {
	// 工具返回的 JSON 格式: {"videos":[...],"total":N}
	// assistant 消息可能以文本开头（如 "为你推荐以下内容：\n"），后面跟着 JSON
	// 策略：找到第一个 { 然后截取到对应的 }
	start := strings.Index(content, "{")
	if start < 0 {
		return
	}
	// 找到匹配的闭合 }
	depth := 0
	end := start
	for i := start; i < len(content); i++ {
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}
	if end <= start {
		return
	}

	var data struct {
		Videos []struct {
			VideoID      int32    `json:"videoID"`
			AuthorID     int32    `json:"authorID"`
			AuthorName   string   `json:"authorName"`
			AuthorAvatar string   `json:"authorAvatar"`
			Title        string   `json:"title"`
			Url          string   `json:"url"`
			LikeNum      int32    `json:"likeNum"`
			CommentNum   int32    `json:"commentNum"`
			CreatedAt    string   `json:"createdAt"`
			Tags         []string `json:"tags"`
		} `json:"videos"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal([]byte(content[start:end]), &data); err != nil {
		return
	}
	if len(data.Videos) == 0 {
		return
	}
	for _, v := range data.Videos {
		resp.Videos = append(resp.Videos, &agentpb.VideoItem{
			VideoID:      v.VideoID,
			AuthorID:     v.AuthorID,
			AuthorName:   v.AuthorName,
			AuthorAvatar: v.AuthorAvatar,
			Title:        v.Title,
			Url:          v.Url,
			LikeNum:      v.LikeNum,
			CommentNum:   v.CommentNum,
			CreatedAt:    v.CreatedAt,
			Tags:         v.Tags,
		})
	}
	resp.Total = data.Total
}
