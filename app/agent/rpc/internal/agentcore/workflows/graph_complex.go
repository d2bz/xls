package workflows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	complexDefaultMaxSteps    = 6
	complexDefaultMaxLoopIter = 3
)

// RunComplexTaskV2 通过 Plan-Execute-Replan 模式执行复杂任务。
// 使用 RunComplexTaskInputV2 作为输入，Slots 包含所有精确参数。
func RunComplexTaskV2(
	ctx context.Context,
	toolChatModel model.ToolCallingChatModel,
	tools []tool.BaseTool,
	input *RunComplexTaskInputV2,
	maxSteps, maxLoop int,
) (*WorkflowResult, error) {
	if maxSteps <= 0 {
		maxSteps = complexDefaultMaxSteps
	}
	if maxLoop <= 0 {
		maxLoop = complexDefaultMaxLoopIter
	}

	task := input.Task
	slots := task.Slots
	if slots == nil {
		slots = &TaskSlot{}
	}

	// 构建 MCP 工具 section
	var mcpToolsSection string
	if len(input.MCPToolsInfo) > 0 {
		for _, t := range input.MCPToolsInfo {
			mcpToolsSection += fmt.Sprintf("- %s: %s\n", t.Name, t.Desc)
		}
		mcpToolsSection = "\n\n## MCP 视频分析工具（可选调用）\n" + mcpToolsSection +
			"\n优先使用确定性的业务工具；需要深度分析视频内容时，使用 MCP 视频分析工具"
	}

	// 1. 创建 Planner
	planner, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: toolChatModel,
		GenInputFn:           genComplexPlannerInput,
	})
	if err != nil {
		return &WorkflowResult{
			ResultType: ResultTypeText,
			Text:       "抱歉，任务规划失败：" + err.Error(),
		}, nil
	}

	// 构建 Executor prompt 模板（包含精确参数）
	executorPromptTpl := buildComplexExecutorPromptTemplate(slots, mcpToolsSection)

	// 2. 创建 Executor
	executor, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model:         toolChatModel,
		ToolsConfig:   adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools}},
		MaxIterations: maxSteps,
		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			planJSON, _ := json.Marshal(in.Plan)
			return executorPromptTpl.Format(ctx, map[string]any{
				"input":           formatComplexUserInputFromMsg(in.UserInput),
				"plan":           string(planJSON),
				"executed_steps":  formatComplexStepsStr(in.ExecutedSteps),
				"step":           in.Plan.FirstStep(),
			})
		},
	})
	if err != nil {
		return &WorkflowResult{
			ResultType: ResultTypeText,
			Text:       "抱歉，任务执行器初始化失败：" + err.Error(),
		}, nil
	}

	// 3. 创建 Replanner
	replanner, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: toolChatModel,
		GenInputFn: genComplexReplannerInput,
	})
	if err != nil {
		return &WorkflowResult{
			ResultType: ResultTypeText,
			Text:       "抱歉，任务重规划失败：" + err.Error(),
		}, nil
	}

	// 4. 创建 Plan-Execute Agent
	agent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planner,
		Executor:      executor,
		Replanner:     replanner,
		MaxIterations: maxLoop,
	})
	if err != nil {
		return &WorkflowResult{
			ResultType: ResultTypeText,
			Text:       "抱歉，任务规划器创建失败：" + err.Error(),
		}, nil
	}

	// 5. 运行 Agent
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	userInput := buildComplexUserInput(slots, task.Query)
	iter := runner.Run(ctx, []*schema.Message{schema.UserMessage(userInput)})

	var lastAnswer string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return &WorkflowResult{
				ResultType: ResultTypeText,
				Text:       "抱歉，任务执行出错：" + event.Err.Error(),
			}, nil
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			if msg, _ := event.Output.MessageOutput.GetMessage(); msg != nil && msg.Content != "" {
				lastAnswer = msg.Content
			}
		}
	}

	if lastAnswer == "" {
		lastAnswer = "抱歉，这个问题比较复杂，AI 暂时无法准确回答。你可以尝试换个方式问我，或者联系管理员。"
	}
	return &WorkflowResult{
		ResultType: ResultTypeText,
		Text:       lastAnswer,
	}, nil
}

// ExecVideoAnalysisAssistantV2 执行视频分析任务。
// 当 Slots.VideoID > 0 时，走强制 MCP 调用路径（直接传显式 video_id 参数）。
// 当没有明确 VideoID 时，走通用的 Plan-Execute 路径。
func ExecVideoAnalysisAssistantV2(
	ctx context.Context,
	toolChatModel model.ToolCallingChatModel,
	tools []tool.BaseTool,
	input *RunComplexTaskInputV2,
	maxSteps, maxLoop int,
) (*WorkflowResult, error) {
	slots := input.Task.Slots
	if slots != nil && slots.VideoID > 0 {
		return runVideoAnalysisWithVideoID(ctx, toolChatModel, tools, input)
	}
	return RunComplexTaskV2(ctx, toolChatModel, tools, input, maxSteps, maxLoop)
}

// runVideoAnalysisWithVideoID 强制用显式 VideoID 调用 MCP 工具，直接传 video_id 参数。
// 不走通用的 Plan-Execute，而是直接调 MCP 工具获取视频内容，再让 LLM 总结。
func runVideoAnalysisWithVideoID(
	ctx context.Context,
	toolChatModel model.ToolCallingChatModel,
	tools []tool.BaseTool,
	input *RunComplexTaskInputV2,
) (*WorkflowResult, error) {
	slots := input.Task.Slots
	videoID := slots.VideoID

	// 1. 找到 MCP 工具
	var mcpTool tool.InvokableTool
	for _, t := range tools {
		if inv, ok := t.(tool.InvokableTool); ok {
			info, err := t.Info(ctx)
			if err != nil {
				continue
			}
			// MCP 工具通过 ToolInfo 区分（mcpToolsInfo 中有记录）
			for _, mcpInfo := range input.MCPToolsInfo {
				if info.Name == mcpInfo.Name {
					mcpTool = inv
					break
				}
			}
			if mcpTool != nil {
				break
			}
		}
	}

	var videoContent string
	if mcpTool != nil {
		// 2. 直接用显式 video_id 调用 MCP 工具
		params := fmt.Sprintf(`{"video_id":%d}`, videoID)
		result, err := mcpTool.InvokableRun(ctx, params)
		if err != nil {
			return &WorkflowResult{
				ResultType: ResultTypeText,
				Text:       fmt.Sprintf("抱歉，获取视频内容失败：%v", err),
			}, nil
		}
		videoContent = result
	} else {
		// 没有 MCP 工具时，构造一个说明性的提示让 LLM 知道需要分析
		videoContent = fmt.Sprintf("用户请求分析视频 ID=%d，但当前无可用的视频内容分析工具。", videoID)
	}

	// 3. 让 LLM 根据视频内容生成总结
	analysisPrompt := fmt.Sprintf(`你是一个专业的视频内容分析助手。以下是视频 ID=%d 的内容信息：

%s

请根据上述内容，为用户提供一段简洁、有价值的分析总结。要求：
1. 总结视频的主要内容或亮点
2. 如果有配乐、BGM、文字内容、场景等特色，一并指出
3. 语言简洁友好，用中文回复
4. 长度控制在 200 字以内
5. 不要编造内容，如果信息不足如实告知用户`, videoID, videoContent)

	resp, err := toolChatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(analysisPrompt),
	})
	if err != nil {
		return &WorkflowResult{
			ResultType: ResultTypeText,
			Text:       "抱歉，分析过程出现错误：" + err.Error(),
		}, nil
	}

	text := resp.Content
	if text == "" {
		text = "抱歉，暂时无法获取该视频的分析结果，请稍后再试。"
	}
	return &WorkflowResult{
		ResultType: ResultTypeText,
		Text:       text,
	}, nil
}

// buildComplexUserInput 构建复杂任务的输入文本，包含精确参数。
func buildComplexUserInput(slots *TaskSlot, query string) string {
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
		preciseParams += fmt.Sprintf("- 数量限制: %d\n", slots.Limit)
	}
	if slots.Sort != "" {
		preciseParams += fmt.Sprintf("- 排序方式: %s\n", slots.Sort)
	}

	if preciseParams != "" {
		return fmt.Sprintf("## 精确参数（来自用户请求，请务必使用）\n%s\n## 用户查询\n%s", preciseParams, query)
	}
	return fmt.Sprintf("## 用户查询\n%s", query)
}

// buildComplexExecutorPromptTemplate 构建 Executor 的 prompt 模板，包含精确参数供 LLM 工具调用使用。
func buildComplexExecutorPromptTemplate(slots *TaskSlot, mcpToolsSection string) prompt.ChatTemplate {
	var baseTools string
	if mcpToolsSection != "" {
		baseTools = `你可以调用以下工具完成任务：
- search_video: 搜索视频
- get_hot_videos: 获取热门视频
- get_user_info: 查询用户信息
- get_follow_list: 查询关注列表
- get_fans_list: 查询粉丝列表` + mcpToolsSection
	} else {
		baseTools = `你可以调用以下工具完成任务：
- search_video: 搜索视频
- get_hot_videos: 获取热门视频
- get_user_info: 查询用户信息
- get_follow_list: 查询关注列表
- get_fans_list: 查询粉丝列表`
	}

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
		preciseParams += fmt.Sprintf("- 数量限制: %d\n", slots.Limit)
	}
	if slots.Sort != "" {
		preciseParams += fmt.Sprintf("- 排序方式: %s\n", slots.Sort)
	}

	var fullPrompt string
	if preciseParams != "" {
		fullPrompt = fmt.Sprintf(`你是一个勤奋的执行者。严格按计划执行当前步骤，并在执行后汇报结果。

## 你的工具
%s

## 精确参数（来自用户请求，调用工具时请务必使用）
%s## 执行原则
- 每次只执行一个步骤
- 调用工具时提供准确的参数（使用上述精确参数中的值）
- 如果上一步骤已有结果，本步骤可以利用该结果
- 执行完成后，用一段话描述执行结果

## 当前任务
{input}

## 计划
{plan}

## 已完成步骤及结果
{executed_steps}

## 当前步骤
{step}

请执行当前步骤。`, baseTools, preciseParams)
	} else {
		fullPrompt = fmt.Sprintf(`你是一个勤奋的执行者。严格按计划执行当前步骤，并在执行后汇报结果。

## 你的工具
%s

## 执行原则
- 每次只执行一个步骤
- 调用工具时提供准确的参数
- 如果上一步骤已有结果，本步骤可以利用该结果
- 执行完成后，用一段话描述执行结果

## 当前任务
{input}

## 计划
{plan}

## 已完成步骤及结果
{executed_steps}

## 当前步骤
{step}

请执行当前步骤。`, baseTools)
	}

	return prompt.FromMessages(schema.FString,
		schema.SystemMessage(fullPrompt),
	)
}

// 以下是 Plan-Execute 的 prompt 模板

var complexPlannerPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`你是一个短视频平台任务规划助手。

## 重要限制
- 每次规划最多规划 5 个步骤
- 如果 3 步内能完成就完成，不要过度规划
- 优先使用确定性工具，少用探索性工具
- 无法确定用户意图时，给出友好回复，不要强行规划

## 你的职责
分析用户的请求，将复杂任务拆解为清晰、可执行的步骤序列。

## 任务类型与拆解策略
### 查视频类
- 搜索视频：确定关键词 → 调用搜索工具 → 整理结果
- 热门视频：获取热榜列表 → 按需排序/筛选

### 查用户类
- 用户信息：获取用户ID → 调用用户信息工具
- 关注列表：获取用户关注列表 → 批量获取详情

### 分析类（多步骤）
- 找出最火视频：获取关注列表 → 获取每个关注用户的视频 → 获取视频热度数据 → 排序
- 趋势分析：获取热榜 → 分析内容方向 → 总结规律

## 拆解要求
- 每个步骤必须是原子操作，对应一个工具调用
- 步骤之间有明确的依赖关系和执行顺序
- 简单查询（如打招呼）只需一步：直接回答

## 输出格式
生成一个 JSON 对象，包含 steps 数组，每个步骤是字符串描述。
示例：
{"steps": ["调用 get_hot_videos 获取当前热门视频列表", "分析热度数据找出前3名"]}`),
	schema.MessagesPlaceholder("input", false),
)

var complexReplannerPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`你负责评估任务完成进度，决定是结束还是继续。

## 重要限制
- 不要过度规划，简单任务快速结束
- 已达到 3 次循环时强制结束

## 决策规则
### 结束任务（COMPLETE）
当满足以下任一条件时，调用 respond 工具给出最终回答：
- 用户的问题已经得到完整解答
- 已收集到所有需要的信息
- 已达到合理的执行步数上限

### 继续执行（CONTINUE）
当满足以下任一条件时，调用 plan 工具制定后续计划：
- 还有关键信息未获取
- 需要多步聚合分析（如：找热度最高的视频需要先获取多个视频的点赞数）
- 上一步骤结果不完整或需要补充

## 历史记录格式
已完成步骤及结果：
{executed_steps}

## 输出要求
- respond 时：给出完整、友好的最终回答，用中文
- plan 时：只列出剩余未完成的步骤，省略已完成的`),
	schema.UserMessage(`## 任务目标
{input}

## 原始计划
{plan}

## 已完成步骤及结果
{executed_steps}`),
)

func genComplexPlannerInput(ctx context.Context, userInput []adk.Message) ([]adk.Message, error) {
	return complexPlannerPrompt.Format(ctx, map[string]any{"input": userInput})
}

func genComplexReplannerInput(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
	planJSON, _ := json.Marshal(in.Plan)
	return complexReplannerPrompt.Format(ctx, map[string]any{
		"input":          formatComplexUserInputFromMsg(in.UserInput),
		"plan":           string(planJSON),
		"executed_steps": formatComplexStepsStr(in.ExecutedSteps),
	})
}

func formatComplexUserInputFromMsg(msgs []adk.Message) string {
	if len(msgs) == 0 {
		return ""
	}
	return msgs[len(msgs)-1].Content
}

func formatComplexStepsStr(steps []planexecute.ExecutedStep) string {
	var result string
	for _, s := range steps {
		result += fmt.Sprintf("步骤: %s\n结果: %s\n\n", s.Step, s.Result)
	}
	return result
}
