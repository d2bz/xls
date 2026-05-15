package config

import (
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	UserRPC    zrpc.RpcClientConf
	VideoRPC   zrpc.RpcClientConf
	LikeRPC    zrpc.RpcClientConf
	FollowRPC  zrpc.RpcClientConf
	CommentRPC zrpc.RpcClientConf
	Agent      AgentConfig
	Milvus     MilvusConfig
}

type MilvusConfig struct {
	Address      string
	Collection   string
	SearchTopK   int
	SearchNProbe int
}

type AgentConfig struct {
	Model      ModelConfig
	Session    SessionConfig
	MaxIter    int
	Skill      SkillConfig
	Middleware MiddlewareConfig
	VideoMCP   VideoMCPConfig
}

// MiddlewareConfig holds configuration for ADK ChatModelAgentMiddleware.
type MiddlewareConfig struct {
	Summarization SummarizationConfig
	Reduction     ReductionConfig
}

// SummarizationConfig configures the summarization middleware.
// When conversation token count exceeds TriggerTokens, the middleware
// compresses history by generating a summary via LLM.
type SummarizationConfig struct {
	Enabled        bool   // 是否启用，默认 false
	TriggerTokens  int    // 触发摘要的 token 阈值，默认 80000
	TriggerMessages int   // 触发摘要的消息数阈值，默认 0（不按消息数触发）
	PreserveTokens int    // 保留用户消息的 token 上限，默认 TriggerTokens/3
	TranscriptPath string // 完整对话记录文件路径，用于 LLM 参考原始上下文
	Instruction    string // 自定义摘要指令，不设置则使用内置默认
}

// ReductionConfig configures the tool reduction middleware.
// It manages tool outputs in two phases:
//  1. Truncation: truncates overlong tool results, offloads to backend
//  2. Clear: clears historical tool results when token budget is tight
type ReductionConfig struct {
	Enabled          bool  // 是否启用，默认 false
	MaxLengthTrunc   int   // 单次工具输出截断阈值，默认 50000 字符
	MaxTokensClear   int64 // 触发工具结果清理的 token 上限，默认 160000
	RootDir          string // 截断/清理内容的存储根目录，默认 /tmp
}

type ModelConfig struct {
	Type   string // "ark" | "openai"
	Ark    ArkModelConfig
	OpenAI OpenAIModelConfig
}

type ArkModelConfig struct {
	APIKey  string
	Model   string
	BaseURL string
	Region  string
}

type OpenAIModelConfig struct {
	APIKey  string
	Model   string
	BaseURL string
}

type SessionConfig struct {
	Type       string // "memory" | "mysql"
	MySQL      MySQLSessionConfig
	MaxHistory int // 最大保留消息数，默认 50
}

type MySQLSessionConfig struct {
	DSN          string // e.g. "user:pass@tcp(localhost:3306)/db"
	MaxIdleConns int
	MaxOpenConns int
}

type SkillConfig struct {
	Dir string // e.g. "internal/agent/skills"
}

// VideoMCPConfig holds configuration for the aigroup-video-mcp server.
type VideoMCPConfig struct {
	Enabled     bool   // 是否启用视频分析 MCP 工具
	Transport   string // "stdio" | "sse"，默认 "stdio"
	Address     string // SSE 模式下 MCP 服务器地址，如 "http://localhost:3001/sse"
	APIKey      string // DashScope API Key，stdio 模式下通过环境变量传递
	ToolFilters []string // 只加载指定工具名，空表示加载全部
}
