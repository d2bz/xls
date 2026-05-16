package router

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"xls/app/agent/rpc/internal/agentcore/workflows"
)

// supervisorNodeFactory 工厂函数，创建 Supervisor Lambda Node。
// 该 Node 调用 LLM 做意图分类，将结果写入 AgentState，返回 intent 字符串供 Branch 使用。
//
// Node 签名: *RouterInput → string
//   输入: RouterInput（包含用户查询和所有精确参数）
//   输出: intent 字符串 (作为 Branch 的输入)
func supervisorNodeFactory(
	chatModel model.ChatModel,
	prompt string,
) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, input *RouterInput) (string, error) {
		state := &AgentState{
			Slots: &workflows.TaskSlot{},
		}

		// 将 RouterInput 中的精确参数合并到 State.Slots
		// 注意：这些参数来自外部精确请求，优先级最高
		state.Slots = &workflows.TaskSlot{
			UserID:   input.UserID,
			VideoID:  input.VideoID,
			Keyword:  input.Keyword,
			Page:     input.Page,
			Limit:    input.PageSize,
		}

		// 构建消息
		userContent := fmt.Sprintf("用户查询: %s\n请分析上述查询，返回意图和参数。", input.Query)
		msgs := []*schema.Message{
			schema.SystemMessage(prompt),
			schema.UserMessage(userContent),
		}

		// 调用 LLM
		resp, err := chatModel.Generate(ctx, msgs)
		if err != nil {
			_ = compose.ProcessState[*AgentState](ctx, func(ctx context.Context, s *AgentState) error {
				s.Error = err.Error()
				s.Intent = workflows.IntentFallback
				s.Slots = &workflows.TaskSlot{}
				s.Confidence = 0
				return nil
			})
			return "", fmt.Errorf("supervisor call failed: %w", err)
		}

		// 解析结果到 State
		state.Reason = parseSupervisorToState(resp.Content, state)

		// 如果解析失败，标记为 fallback
		if state.Intent == "" {
			state.Intent = workflows.IntentFallback
			state.Slots = &workflows.TaskSlot{}
			state.Confidence = 0
		}

		// 置信度兜底
		if state.Confidence < MinConfidence {
			state.Intent = workflows.IntentFallback
		}

		// 将结果写入 Graph State
		_ = compose.ProcessState[*AgentState](ctx, func(ctx context.Context, s *AgentState) error {
			s.Intent = state.Intent
			s.Slots = state.Slots
			s.Confidence = state.Confidence
			s.Reason = state.Reason
			return nil
		})

		return string(state.Intent), nil
	})
}

// parseSupervisorToState 解析 LLM 输出，将结果写入 AgentState，返回 reason 字符串。
func parseSupervisorToState(content string, state *AgentState) string {
	content = trimCodeBlock(content)

	var result struct {
		Tasks   []workflows.Task `json:"tasks"`
		IsMulti bool            `json:"is_multi"`
		Reason  string          `json:"reason"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		state.Intent = workflows.IntentFallback
		state.Slots = &workflows.TaskSlot{}
		state.Confidence = 0
		return fmt.Sprintf("解析失败: %s", err.Error())
	}

	if len(result.Tasks) == 0 {
		state.Intent = workflows.IntentFallback
		state.Slots = &workflows.TaskSlot{}
		state.Confidence = 0
		return "无任务"
	}

	// 取置信度最高的任务
	best := result.Tasks[0]
	for _, t := range result.Tasks[1:] {
		if t.Confidence > best.Confidence {
			best = t
		}
	}

	state.Intent = best.Intent
	state.Slots = best.Slots
	state.Confidence = best.Confidence
	if state.Slots == nil {
		state.Slots = &workflows.TaskSlot{}
	}

	// 置信度兜底
	if state.Confidence < MinConfidence {
		state.Intent = workflows.IntentFallback
	}

	return result.Reason
}
