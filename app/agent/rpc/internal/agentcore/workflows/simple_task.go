package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
// input: 任务输入，包含完整的 Task 信息，其中 Slots 包含所有精确参数。
// maxSteps: 最大循环步数，默认 4（约 2 轮工具调用）。
func RunSimpleTask(
	ctx context.Context,
	toolChatModel model.ToolCallingChatModel,
	tools []tool.BaseTool,
	input *RunSimpleTaskInput,
	maxSteps int,
) (*WorkflowResult, error) {
	if maxSteps <= 0 {
		maxSteps = simpleTaskDefaultMaxSteps
	}

	task := input.Task
	slots := task.Slots
	if slots == nil {
		slots = &TaskSlot{}
	}

	// 判断意图是否属于视频推荐/搜索类
	isVideoListIntent := isVideoListIntent(task.Intent)

	// 构建 system prompt，将精确参数注入供 LLM 工具调用时使用
	systemPrompt := buildSimpleTaskSystemPrompt(task.Query, slots, isVideoListIntent)

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
		return &WorkflowResult{
			ResultType: ResultTypeText,
			Text:       "抱歉，处理请求时遇到问题：" + err.Error(),
		}, nil
	}

	// 运行 agent
	out, err := agent.Generate(ctx, []*schema.Message{
		schema.UserMessage(task.Query),
	})
	if err != nil {
		return &WorkflowResult{
			ResultType: ResultTypeText,
			Text:       "抱歉，处理请求时遇到问题：" + err.Error(),
		}, nil
	}

	if out == nil || out.Content == "" {
		return &WorkflowResult{
			ResultType: ResultTypeText,
			Text:       "抱歉，暂时无法处理你的请求，请稍后再试。",
		}, nil
	}

	// 尝试从回复中提取工具返回的 JSON 视频数据
	var videos []*VideoItem
	var total int64
	if isVideoListIntent {
		videos, total = extractVideosFromReActOutput(out.Content)
	}

	result := &WorkflowResult{
		Text:   out.Content,
		Videos: videos,
		Total:  total,
	}
	if len(videos) > 0 {
		result.ResultType = ResultTypeVideoList
	} else {
		result.ResultType = ResultTypeText
	}
	return result, nil
}

// isVideoListIntent 判断意图是否属于需要返回视频列表的类型。
func isVideoListIntent(intent Intent) bool {
	switch intent {
	case IntentVideoSearch, IntentRecommend, IntentRecommendSemantic:
		return true
	default:
		return false
	}
}

// extractVideosFromReActOutput 从 ReAct agent 的输出文本中解析 JSON 视频数据。
// ReAct 回复格式：友好的中文文本 + 末尾嵌入了 {"videos":[...],"total":N} 元数据。
func extractVideosFromReActOutput(content string) ([]*VideoItem, int64) {
	// 在文本中查找 JSON 对象（最后一个 { 开头到 } 结尾的块）
	start := strings.LastIndex(content, "{")
	if start < 0 {
		return nil, 0
	}
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
		return nil, 0
	}

	var raw struct {
		Videos []struct {
			ID         uint64   `json:"id"`
			Title      string   `json:"title"`
			AuthorID   uint64   `json:"author_id"`
			AuthorName string   `json:"author_name"`
			LikeCount  int64    `json:"like_count"`
			Duration   int      `json:"duration"`
			Tags       []string `json:"tags"`
		} `json:"videos"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal([]byte(content[start:end]), &raw); err != nil {
		return nil, 0
	}
	if len(raw.Videos) == 0 {
		return nil, 0
	}

	items := make([]*VideoItem, 0, len(raw.Videos))
	for _, v := range raw.Videos {
		items = append(items, &VideoItem{
			VideoID:    int32(v.ID),
			AuthorID:   int32(v.AuthorID),
			AuthorName: v.AuthorName,
			Title:      v.Title,
			LikeNum:    int32(v.LikeCount),
			Tags:       v.Tags,
		})
	}
	return items, raw.Total
}

// buildSimpleTaskSystemPrompt 构建 ReAct Agent 的 system prompt。
// 将精确参数注入 prompt，使 LLM 能够正确调用工具。
func buildSimpleTaskSystemPrompt(query string, slots *TaskSlot, isVideoListIntent bool) string {
	// 构建精确参数 section
	var preciseParams string
	if slots.UserID > 0 {
		preciseParams += fmt.Sprintf("- 用户ID: %d\n", slots.UserID)
	}
	if slots.VideoID > 0 {
		preciseParams += fmt.Sprintf("- 视频ID: %d\n", slots.VideoID)
	}
	if slots.Keyword != "" {
		preciseParams += fmt.Sprintf("- 搜索关键词: %s\n", slots.Keyword)
	}
	if slots.Page > 0 {
		preciseParams += fmt.Sprintf("- 页码: %d\n", slots.Page)
	}
	if slots.Limit > 0 {
		preciseParams += fmt.Sprintf("- 每页数量: %d\n", slots.Limit)
	}
	if slots.Sort != "" {
		preciseParams += fmt.Sprintf("- 排序方式: %s\n", slots.Sort)
	}

	toolsSection := buildSimpleTaskToolsSection()

	// 构建回复格式要求
	var formatInstruction string
	if isVideoListIntent {
		formatInstruction = `## 回复格式要求
重要：你必须在回复末尾嵌入 JSON 元数据，供前端渲染视频列表。
格式如下（必须严格遵守）：
{"videos":[{"id":123,"title":"视频标题","author_id":456,"author_name":"作者名","like_count":1000,"duration":180,"tags":["tag1","tag2"]}],"total":10}

规则：
- JSON 必须紧跟在友好文本之后，可以换行但必须完整
- videos 数组中的每个对象必须包含: id, title, author_id, author_name
- 如果没有搜索结果，total 为 0，videos 为空数组
- 不要在 JSON 前后添加任何其他文字或解释
`
	} else {
		formatInstruction = `## 回复格式要求
- 直接给出简洁、友好的中文回复。
- 不要过度解释或重复用户的话。
`
	}

	if preciseParams != "" {
		return fmt.Sprintf(`你是一个简洁高效的短视频平台任务助手。

## 你的工具
%s

## 精确参数（来自用户请求，请务必使用）
%s## 执行原则
- 根据用户查询，选择最合适的工具调用。
- 优先使用确定性工具，少用探索性工具。
- 每次只调用一个工具。
- 调用工具时，请使用上述精确参数中的值（如 user_id、video_id、keyword、page 等）。
- 如果工具返回空数据，友好提示用户并给出建议。
- 不要过度解释或重复用户的话，直接给出有用信息。
%s

## 当前任务
%s`, toolsSection, preciseParams, formatInstruction, query)
	}

	return fmt.Sprintf(`你是一个简洁高效的短视频平台任务助手。

## 你的工具
%s

## 执行原则
- 根据用户查询，选择最合适的工具调用。
- 优先使用确定性工具，少用探索性工具。
- 每次只调用一个工具。
- 如果工具返回空数据，友好提示用户并给出建议。
- 不要过度解释或重复用户的话，直接给出有用信息。
%s

## 当前任务
%s`, toolsSection, formatInstruction, query)
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
