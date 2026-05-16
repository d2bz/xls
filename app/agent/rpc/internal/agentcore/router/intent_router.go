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
// Graph 签名: *RouterInput → *WorkflowResult
//   输入: RouterInput（包含用户查询和所有精确参数）
//   输出: WorkflowResult（包含回复文本、视频列表、结果类型）
func NewIntentRouterGraph(
	ctx context.Context,
	cfg RouterGraphConfig,
) (compose.Runnable[*RouterInput, *workflows.WorkflowResult], error) {
	// 1. 初始化 State 生成器
	genState := func(ctx context.Context) *AgentState {
		return &AgentState{
			Slots: &workflows.TaskSlot{},
		}
	}

	// 2. 创建顶层 Graph: *RouterInput → *WorkflowResult
	g := compose.NewGraph[*RouterInput, *workflows.WorkflowResult](
		compose.WithGenLocalState(genState),
	)

	// 3. Supervisor Node: *RouterInput → intent string
	// Supervisor 接收完整的 RouterInput，从中提取 Query 用于意图分类，
	// 同时将精确参数合并到 State.Slots 中，供下游节点使用。
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
	// 所有意图节点都从 START 边接收 *RouterInput（包含所有精确参数）
	// 同时通过 ProcessState 读取 State 中的分类结果（intent, slots, confidence）
	// 参数通过边的数据流（Field Mapping）传递，保证类型安全

	// 5.1 simple_task: 合并 video_search、recommend、user_relation 三个简单意图，统一由 ReAct 处理
	simpleTaskNode := compose.InvokableLambda(func(ctx context.Context, input *RouterInput) (*workflows.WorkflowResult, error) {
		var intent workflows.Intent
		var confidence float64
		var slots *workflows.TaskSlot
		_ = compose.ProcessState[*AgentState](ctx, func(ctx context.Context, s *AgentState) error {
			intent = s.Intent
			confidence = s.Confidence
			slots = s.Slots
			return nil
		})

		// 构建 Task，将 RouterInput 中的精确参数与 State.Slots 合并
		// RouterInput 参数优先（来自外部精确请求），State.Slots 兜底（来自 LLM 解析）
		task := &workflows.Task{
			Query:      input.Query,
			Intent:     intent,
			Slots:      mergeSlots(input, slots),
			Confidence: confidence,
		}

		runInput := &workflows.RunSimpleTaskInput{
			Task: task,
		}
		result, err := workflows.RunSimpleTask(ctx, cfg.ToolChatModel, cfg.Tools, runInput, cfg.MaxSteps)
		if err != nil {
			return &workflows.WorkflowResult{
				ResultType: workflows.ResultTypeText,
				Text:       "抱歉，处理请求时遇到问题：" + err.Error(),
			}, nil
		}
		return result, nil
	})
	if err := g.AddLambdaNode("simple_task", simpleTaskNode); err != nil {
		return nil, fmt.Errorf("add simple_task node failed: %w", err)
	}

	// 5.2 video_analysis (Plan-Execute 多轮助手，支持动态工具选择 + MCP 工具)
	videoAnalysisNode := compose.InvokableLambda(func(ctx context.Context, input *RouterInput) (*workflows.WorkflowResult, error) {
		var intent workflows.Intent
		var confidence float64
		var slots *workflows.TaskSlot
		_ = compose.ProcessState[*AgentState](ctx, func(ctx context.Context, s *AgentState) error {
			intent = s.Intent
			confidence = s.Confidence
			slots = s.Slots
			return nil
		})

		task := &workflows.Task{
			Query:      input.Query,
			Intent:     intent,
			Slots:      mergeSlots(input, slots),
			Confidence: confidence,
		}

		runInput := &workflows.RunComplexTaskInputV2{
			Task:         task,
			MCPToolsInfo: cfg.MCPToolsInfo,
		}
		result, err := workflows.ExecVideoAnalysisAssistantV2(ctx, cfg.ToolChatModel, cfg.Tools, runInput, cfg.MaxSteps, cfg.MaxLoopIter)
		if err != nil {
			logx.Errorf("[video_analysis] plan-execute failed: %v", err)
			return &workflows.WorkflowResult{
				ResultType: workflows.ResultTypeText,
				Text:       "抱歉，视频分析处理失败：" + err.Error(),
			}, nil
		}
		return result, nil
	})
	if err := g.AddLambdaNode("video_analysis", videoAnalysisNode); err != nil {
		return nil, fmt.Errorf("add video_analysis node failed: %w", err)
	}

	// 5.3 complex (Plan-Execute-Replan)
	complexNode := compose.InvokableLambda(func(ctx context.Context, input *RouterInput) (*workflows.WorkflowResult, error) {
		var intent workflows.Intent
		var confidence float64
		var slots *workflows.TaskSlot
		_ = compose.ProcessState[*AgentState](ctx, func(ctx context.Context, s *AgentState) error {
			intent = s.Intent
			confidence = s.Confidence
			slots = s.Slots
			return nil
		})

		task := &workflows.Task{
			Query:      input.Query,
			Intent:     intent,
			Slots:      mergeSlots(input, slots),
			Confidence: confidence,
		}

		runInput := &workflows.RunComplexTaskInputV2{
			Task:         task,
			MCPToolsInfo: cfg.MCPToolsInfo,
		}
		result, err := workflows.RunComplexTaskV2(ctx, cfg.ToolChatModel, cfg.Tools, runInput, cfg.MaxSteps, cfg.MaxLoopIter)
		if err != nil {
			logx.Errorf("[complex] plan-execute failed: %v", err)
			return &workflows.WorkflowResult{
				ResultType: workflows.ResultTypeText,
				Text:       "抱歉，复杂任务处理失败：" + err.Error(),
			}, nil
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

	videoSemanticRecommendNode := compose.InvokableLambda(func(ctx context.Context, input *RouterInput) (*workflows.WorkflowResult, error) {
		var slots *workflows.TaskSlot
		_ = compose.ProcessState[*AgentState](ctx, func(ctx context.Context, s *AgentState) error {
			slots = s.Slots
			return nil
		})

		mergedSlots := mergeSlots(input, slots)
		limit := mergedSlots.Limit
		if limit <= 0 {
			limit = 10
		}

		workflowInput := &workflows.SemanticWorkflowInput{
			Query:  input.Query,
			Limit:  limit,
			Dims:   mergedSlots.Dims,
			UserID: mergedSlots.UserID,
		}

		out, err := semanticWorkflow.Invoke(ctx, workflowInput)
		if err != nil {
			logx.Errorf("[video_semantic_recommend] workflow invoke failed: %v", err)
			return &workflows.WorkflowResult{
				ResultType: workflows.ResultTypeText,
				Text:       "推荐处理失败，请稍后再试。",
			}, nil
		}
		return out, nil
	})
	if err = g.AddLambdaNode("video_semantic_recommend", videoSemanticRecommendNode); err != nil {
		return nil, fmt.Errorf("add video_semantic_recommend node failed: %w", err)
	}

	// 6. 边连接
	// START → supervisor: 传递完整的 *RouterInput
	if err := g.AddEdge(compose.START, "supervisor"); err != nil {
		return nil, fmt.Errorf("add edge START->supervisor failed: %w", err)
	}

	// 各执行节点: 从 supervisor 边接收 *RouterInput
	// eino Graph 自动将上游输出作为下游输入，无需显式边连接
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

// mergeSlots 将 RouterInput 中的精确参数与 State.Slots 合并。
// RouterInput 参数优先（来自外部精确请求），State.Slots 兜底（来自 LLM 解析）。
func mergeSlots(input *RouterInput, stateSlots *workflows.TaskSlot) *workflows.TaskSlot {
	if stateSlots == nil {
		stateSlots = &workflows.TaskSlot{}
	}

	merged := &workflows.TaskSlot{}

	// UserID: RouterInput 优先
	if input.UserID > 0 {
		merged.UserID = input.UserID
	} else {
		merged.UserID = stateSlots.UserID
	}

	// VideoID: RouterInput 优先
	if input.VideoID > 0 {
		merged.VideoID = input.VideoID
	} else {
		merged.VideoID = stateSlots.VideoID
	}

	// Keyword: RouterInput 优先
	if input.Keyword != "" {
		merged.Keyword = input.Keyword
	} else {
		merged.Keyword = stateSlots.Keyword
	}

	// Page: RouterInput 优先
	if input.Page > 0 {
		merged.Page = input.Page
	} else {
		merged.Page = stateSlots.Page
	}

	// PageSize → Limit: RouterInput 优先
	if input.PageSize > 0 {
		merged.Limit = input.PageSize
	} else {
		merged.Limit = stateSlots.Limit
	}

	// AuthorID, TargetUID, Sort, Dims: 仅从 State.Slots 获取
	merged.AuthorID = stateSlots.AuthorID
	merged.TargetUID = stateSlots.TargetUID
	merged.Sort = stateSlots.Sort
	merged.Dims = stateSlots.Dims

	return merged
}
