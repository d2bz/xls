package workflows

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	simpleTaskDefaultMaxSteps = 4
)

// RunSimpleTask 通过 ReAct 模式执行简单任务。
// 该方法将所有业务工具暴露给 ReAct，ReAct 根据用户意图自主选择工具并生成最终回复。
// 适用于 video_search、recommend、user_relation 等简单意图，每个意图通常只需要一次工具调用。
//
// toolChatModel: 已绑定业务工具的 ToolCallingChatModel。
// tools: 全部业务工具列表（6个：search_video、get_hot_videos、get_user_info、get_follow_list、get_fans_list、get_videos_by_dimensions）。
// input: 任务输入，包含用户查询和用户ID。
// maxSteps: 最大循环步数，默认 4（约 2 轮工具调用）。
func RunSimpleTask(
	ctx context.Context,
	toolChatModel model.ToolCallingChatModel,
	tools []tool.BaseTool,
	input *RunComplexTaskInput,
	maxSteps int,
) (string, error) {
	if maxSteps <= 0 {
		maxSteps = simpleTaskDefaultMaxSteps
	}

	// 构建业务工具描述 section，供 system prompt 使用
	toolsSection := buildSimpleTaskToolsSection()

	// 构建 system prompt：指导 LLM 使用工具完成任务
	systemPrompt := fmt.Sprintf(`你是一个简洁高效的短视频平台任务助手。

## 你的工具
%s

## 执行原则
- 根据用户查询，选择最合适的工具调用。
- 每次只调用一个工具。
- 工具调用后，根据返回结果生成简洁、友好的中文回复。
- 如果工具返回空数据，友好提示用户并给出建议。
- 不要过度解释或重复用户的话，直接给出有用信息。
- 搜索/推荐类问题，在回复末尾可加一句："输入序号可查看详情，或告诉我其他需求。"

## 当前任务
用户ID: %d
查询: %s`, toolsSection, input.UserID, input.Query)

	// 创建 ReAct agent
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: toolChatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MaxStep: maxSteps,
		MessageModifier: func(ctx context.Context, msgs []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(msgs)+1)
			res = append(res, schema.SystemMessage(systemPrompt))
			res = append(res, msgs...)
			return res
		},
	})
	if err != nil {
		return "", fmt.Errorf("create simple task react agent failed: %w", err)
	}

	// 运行 agent
	out, err := agent.Generate(ctx, []*schema.Message{
		schema.UserMessage(input.Query),
	})
	if err != nil {
		return "", fmt.Errorf("simple task generate failed: %w", err)
	}

	if out == nil || out.Content == "" {
		return "抱歉，暂时无法处理你的请求，请稍后再试。", nil
	}
	return out.Content, nil
}

// buildSimpleTaskToolsSection 构建工具描述文本，供 system prompt 使用。
func buildSimpleTaskToolsSection() string {
	return `- search_video: 根据关键词搜索视频列表。输入：keyword(关键词)、page(页码)、page_size(每页数量)
- get_hot_videos: 获取当前热门视频列表。输入：limit(返回数量)
- get_user_info: 获取指定用户的基本信息。输入：user_id(用户ID)
- get_follow_list: 获取指定用户的关注列表。输入：user_id(用户ID)、page(页码)
- get_fans_list: 获取指定用户的粉丝列表。输入：user_id(用户ID)、page(页码)
- get_videos_by_dimensions: 根据语义维度搜索视频。输入：dims(维度列表，每个维度含name、tags、weight)、limit(数量)、page(页码)`
}
