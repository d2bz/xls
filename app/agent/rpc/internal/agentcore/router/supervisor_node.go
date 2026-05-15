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
// Node 签名: string → string
//   输入: 用户查询 (来自 routerGraphTool，可包含 ## 请求参数 前缀)
//   输出: intent 字符串 (作为 Branch 的输入)
func supervisorNodeFactory(
	chatModel model.ChatModel,
	prompt string,
) *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, query string) (string, error) {
		state := &AgentState{
			Slots: &workflows.TaskSlot{},
		}

		// 解析请求参数前缀（由 routerGraphTool 注入）
		// 格式: ## 请求参数\nkey=value\nkey=value\n\n## 用户查询\nactual_query
		parsedQuery, reqParams := workflows.ParseRequestParams(query)
		if reqParams != nil {
			state.Slots = &workflows.TaskSlot{
				VideoID: reqParams.VideoID,
				Keyword: reqParams.Keyword,
				Page:    reqParams.Page,
				Limit:   reqParams.PageSize,
				UserID:  reqParams.UserID,
			}
		}

		// 构建消息
		userContent := fmt.Sprintf("用户查询: %s\n请分析上述查询，返回意图和参数。", parsedQuery)
		msgs := []*schema.Message{
			schema.SystemMessage(prompt),
			schema.UserMessage(userContent),
		}

		// 调用 LLM
		resp, err := chatModel.Generate(ctx, msgs)
		if err != nil {
			state.Error = err.Error()
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
		state.Reason = parseSupervisorToState(resp.Content, state, parsedQuery)

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
func parseSupervisorToState(content string, state *AgentState, parsedQuery string) string {
	content = trimCodeBlock(content)

	var result struct {
		Tasks   []workflows.Task `json:"tasks"`
		IsMulti bool             `json:"is_multi"`
		Reason  string           `json:"reason"`
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

	// 合并请求参数：request params 优先填充具体字段，LLM 解析结果兜底
	mergeRequestParamsToState(state, parsedQuery)

	// 置信度兜底
	if state.Confidence < MinConfidence {
		state.Intent = workflows.IntentFallback
	}

	return result.Reason
}

// mergeRequestParamsToState 将解析出的请求参数合并到 State.Slots 中。
// 具体字段（video_id, keyword, page, page_size）优先使用请求参数，LLM 结果兜底。
func mergeRequestParamsToState(state *AgentState, query string) {
	if query == "" {
		return
	}
	_, params := workflows.ParseRequestParams(query)
	if params == nil {
		return
	}
	if state.Slots == nil {
		state.Slots = &workflows.TaskSlot{}
	}
	if params.VideoID > 0 {
		state.Slots.VideoID = params.VideoID
	}
	if params.Keyword != "" {
		state.Slots.Keyword = params.Keyword
	}
	if params.Page > 0 {
		state.Slots.Page = params.Page
	}
	if params.PageSize > 0 {
		state.Slots.Limit = params.PageSize
	}
	if params.UserID > 0 {
		state.Slots.UserID = params.UserID
	}
}
