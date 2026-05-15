package router

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"xls/app/agent/rpc/internal/agentcore/workflows"
)

// intentBranchCondition 是 Branch 的条件函数。
// Branch 附加在 supervisor 节点上，接收 supervisor 的输出（intent 字符串）。
// 通过 ProcessState 从 AgentState 中读取意图来做路由决策。
//
// 关键：Branch 条件函数中无法使用闭包变量，必须用 ProcessState。
// 这是 eino state_graph.go 官方示例特别强调的。
func intentBranchCondition(ctx context.Context, _ string) (string, error) {
	var next string
	if err := compose.ProcessState[*AgentState](ctx, func(ctx context.Context, s *AgentState) error {
		// 置信度过低或标记为 fallback → 复杂任务
		if s.Intent == workflows.IntentFallback || s.Confidence < 0.65 {
			next = "complex"
			return nil
		}
		// 根据意图映射到节点名
		switch s.Intent {
		case workflows.IntentVideoSearch, workflows.IntentRecommend, workflows.IntentUserRelation:
			// 三个简单意图统一路由到 simple_task（ReAct 处理）
			next = "simple_task"
		case workflows.IntentVideoAnalysis:
			next = "video_analysis"
		case workflows.IntentRecommendSemantic:
			next = "video_semantic_recommend"
		case workflows.IntentComplexAnalysis, workflows.IntentGeneral:
			next = "complex"
		default:
			next = "complex"
		}
		return nil
	}); err != nil {
		return "", err
	}
	return next, nil
}

// intentBranchFactory 创建意图路由 Branch。
func intentBranchFactory() *compose.GraphBranch {
	return compose.NewGraphBranch(
		intentBranchCondition,
		map[string]bool{
			"simple_task":                true,
			"video_analysis":             true,
			"complex":                    true,
			"video_semantic_recommend":   true,
		},
	)
}
