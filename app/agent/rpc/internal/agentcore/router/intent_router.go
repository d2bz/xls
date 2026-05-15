package router

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
	"xls/app/agent/rpc/internal/agentcore/milvus"
	"xls/app/agent/rpc/internal/agentcore/workflows"
	"xls/pkg/embedding"
)

type RouterGraphConfig struct {
	ChatModel        model.ChatModel
	ToolChatModel    model.ToolCallingChatModel
	Tools            []tool.BaseTool
	SupervisorPrompt string
	MaxSteps         int
	MaxLoopIter      int
	Embedder         *embedding.Embedder
	Milvus           *milvus.Client
	MilvusSearchTopK int
	MCPToolsInfo     []*schema.ToolInfo // MCP 工具的 ToolInfo，动态注入到 prompt 中
}

// NewIntentRouterGraph 创建意图路由 Graph。
// 这是整个 Agent 的核心编排逻辑。
//
// Graph 签名: string → string
//   输入: 用户查询 (来自 START)
//   输出: 最终回答字符串 (流向 END)
func NewIntentRouterGraph(
	ctx context.Context,
	cfg RouterGraphConfig,
) (compose.Runnable[string, string], error) {
	// 1. 初始化 State 生成器
	genState := func(ctx context.Context) *AgentState {
		return &AgentState{
			Slots: &workflows.TaskSlot{},
		}
	}

	// 2. 创建顶层 Graph: string → string
	g := compose.NewGraph[string, string](
		compose.WithGenLocalState(genState),
	)

	// 3. Supervisor Node: query → intent string
	supervisorPrompt := buildSupervisorPrompt(cfg.MCPToolsInfo)
	supervisorNode := supervisorNodeFactory(cfg.ChatModel, supervisorPrompt)
	if err := g.AddLambdaNode("supervisor", supervisorNode); err != nil {
		return nil, fmt.Errorf("add supervisor node failed: %w", err)
	}

	// 4. Branch: 基于 State.Intent 条件路由
	branch := intentBranchFactory()
	if err := g.AddBranch("supervisor", branch); err != nil {
		return nil, fmt.Errorf("add branch failed: %w", err)
	}

	// 5. 意图执行节点
	// 所有意图节点都从 START 边接收 query string（来自 START 的输出）
	// 同时通过 ProcessState 读取 State 中的分类结果（intent, slots, confidence）

	// 5.1 simple_task: 合并 video_search、recommend、user_relation 三个简单意图，统一由 ReAct 处理
	simpleTaskNode := compose.InvokableLambda(func(ctx context.Context, query string) (string, error) {
		var userID uint64
		_ = compose.ProcessState[*AgentState](ctx, func(ctx context.Context, s *AgentState) error {
			userID = s.Slots.UserID
			return nil
		})
		input := &workflows.RunComplexTaskInput{
			Query:  query,
			UserID: userID,
		}
		result, err := workflows.RunSimpleTask(ctx, cfg.ToolChatModel, cfg.Tools, input, cfg.MaxSteps)
		if err != nil {
			return "抱歉，处理请求时遇到问题：" + err.Error(), nil
		}
		return result, nil
	})
	if err := g.AddLambdaNode("simple_task", simpleTaskNode); err != nil {
		return nil, fmt.Errorf("add simple_task node failed: %w", err)
	}

	// 5.2 video_analysis (Plan-Execute 多轮助手，支持动态工具选择 + MCP 工具)
	videoAnalysisNode := compose.InvokableLambda(func(ctx context.Context, query string) (string, error) {
		var userID uint64
		_ = compose.ProcessState[*AgentState](ctx, func(ctx context.Context, s *AgentState) error {
			userID = s.Slots.UserID
			return nil
		})
		input := &workflows.RunComplexTaskInput{
			Query:  query,
			UserID: userID,
		}
		result, err := workflows.ExecVideoAnalysisAssistant(ctx, cfg.ToolChatModel, cfg.Tools, cfg.MCPToolsInfo, input, cfg.MaxSteps, cfg.MaxLoopIter)
		if err != nil {
			logx.Errorf("[video_analysis] plan-execute failed: %v", err)
			return "抱歉，视频分析处理失败：" + err.Error(), nil
		}
		return result, nil
	})
	if err := g.AddLambdaNode("video_analysis", videoAnalysisNode); err != nil {
		return nil, fmt.Errorf("add video_analysis node failed: %w", err)
	}

	// 5.3 complex (Plan-Execute-Replan)
	complexNode := compose.InvokableLambda(func(ctx context.Context, query string) (string, error) {
		var userID uint64
		_ = compose.ProcessState[*AgentState](ctx, func(ctx context.Context, s *AgentState) error {
			userID = s.Slots.UserID
			return nil
		})
		input := &workflows.RunComplexTaskInput{
			Query:  query,
			UserID: userID,
		}
		result, err := workflows.RunComplexTask(ctx, cfg.ToolChatModel, cfg.Tools, cfg.MCPToolsInfo, input, cfg.MaxSteps, cfg.MaxLoopIter)
		if err != nil {
			logx.Errorf("[complex] plan-execute failed: %v", err)
			return "抱歉，复杂任务处理失败：" + err.Error(), nil
		}
		return result, nil
	})
	if err := g.AddLambdaNode("complex", complexNode); err != nil {
		return nil, fmt.Errorf("add complex node failed: %w", err)
	}

	// 5.4 video_semantic_recommend: 先编译 Workflow，再作为 SubGraph 节点接入
	semanticWorkflow, err := workflows.BuildVideoSemanticRecommendWorkflow(ctx, workflows.SemanticRecommendDeps{
		Tools:            cfg.Tools,
		Embedder:         cfg.Embedder,
		Milvus:           cfg.Milvus,
		MilvusSearchTopK: cfg.MilvusSearchTopK,
		ChatModel:        cfg.ChatModel,
	})
	if err != nil {
		return nil, fmt.Errorf("build semantic workflow failed: %w", err)
	}

	videoSemanticRecommendNode := compose.InvokableLambda(func(ctx context.Context, query string) (string, error) {
		var task *workflows.Task
		if err := compose.ProcessState[*AgentState](ctx, func(ctx context.Context, s *AgentState) error {
			task = &workflows.Task{
				Query:     query,
				Intent:    s.Intent,
				Slots:     s.Slots,
				Confidence: s.Confidence,
			}
			return nil
		}); err != nil {
			return "", fmt.Errorf("read state failed: %w", err)
		}

		limit := task.Slots.Limit
		if limit <= 0 {
			limit = 10
		}

		input := &workflows.SemanticWorkflowInput{
			Query:  task.Query,
			Limit:  limit,
			Dims:   task.Slots.Dims,
			UserID: task.Slots.UserID,
		}

		out, err := semanticWorkflow.Invoke(ctx, input)
		if err != nil {
			logx.Errorf("[video_semantic_recommend] workflow invoke failed: %v", err)
			return "", err
		}
		return out.Answer, nil
	})
	if err = g.AddLambdaNode("video_semantic_recommend", videoSemanticRecommendNode); err != nil {
		return nil, fmt.Errorf("add video_semantic_recommend node failed: %w", err)
	}

	// 6. 边连接
	if err := g.AddEdge(compose.START, "supervisor"); err != nil {
		return nil, fmt.Errorf("add edge START->supervisor failed: %w", err)
	}
	if err := g.AddEdge("simple_task", compose.END); err != nil {
		return nil, fmt.Errorf("add edge simple_task->END failed: %w", err)
	}
	if err := g.AddEdge("video_analysis", compose.END); err != nil {
		return nil, fmt.Errorf("add edge video_analysis->END failed: %w", err)
	}
	if err := g.AddEdge("complex", compose.END); err != nil {
		return nil, fmt.Errorf("add edge complex->END failed: %w", err)
	}
	if err := g.AddEdge("video_semantic_recommend", compose.END); err != nil {
		return nil, fmt.Errorf("add edge video_semantic_recommend->END failed: %w", err)
	}

	// 7. 编译
	return g.Compile(ctx, compose.WithMaxRunSteps(50))
}
