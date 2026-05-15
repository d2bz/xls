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

// RunComplexTask 通过 Plan-Execute-Replan 模式执行复杂任务。
// toolChatModel 应该是已经绑定了工具的 ToolCallingChatModel。
// mcpToolsInfo 是 MCP 工具的 ToolInfo 列表，用于动态注入到 prompt 中，
// 让 LLM 知道可以调用哪些 MCP 视频分析工具。
func RunComplexTask(
	ctx context.Context,
	toolChatModel model.ToolCallingChatModel,
	tools []tool.BaseTool,
	mcpToolsInfo []*schema.ToolInfo,
	input *RunComplexTaskInput,
	maxSteps, maxLoop int,
) (string, error) {
	if maxSteps <= 0 {
		maxSteps = complexDefaultMaxSteps
	}
	if maxLoop <= 0 {
		maxLoop = complexDefaultMaxLoopIter
	}

	// 1. 创建 Planner
	planner, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: toolChatModel,
		GenInputFn:         genComplexPlannerInput,
	})
	if err != nil {
		return "", fmt.Errorf("create planner failed: %w", err)
	}

	// 构建动态 executor prompt，注入 MCP 工具列表
	var mcpToolsSection string
	for _, t := range mcpToolsInfo {
		mcpToolsSection += fmt.Sprintf("- %s: %s\n", t.Name, t.Desc)
	}
	if mcpToolsSection != "" {
		mcpToolsSection = "\n\n## MCP 视频分析工具（可选调用）\n" + mcpToolsSection +
			"\n优先使用确定性的业务工具；需要深度分析视频内容时，使用 MCP 视频分析工具"
	}

	// 2. 创建 Executor
	executor, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model:         toolChatModel,
		ToolsConfig: adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools}},
		MaxIterations: maxSteps,
		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			planJSON, _ := json.Marshal(in.Plan)
			promptText := `你是一个勤奋的执行者。严格按计划执行当前步骤，并在执行后汇报结果。

## 你的工具
你可以调用以下工具完成任务：
- search_video: 搜索视频
- get_hot_videos: 获取热门视频
- get_user_info: 查询用户信息
- get_follow_list: 查询关注列表
- get_fans_list: 查询粉丝列表` + mcpToolsSection + `

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

请执行当前步骤。`
			promptTpl := prompt.FromMessages(schema.FString,
				schema.SystemMessage(promptText),
			)
			return promptTpl.Format(ctx, map[string]any{
				"input":          formatComplexUserInputFromMsg(in.UserInput),
				"plan":           string(planJSON),
				"executed_steps": formatComplexStepsStr(in.ExecutedSteps),
				"step":           in.Plan.FirstStep(),
			})
		},
	})
	if err != nil {
		return "", fmt.Errorf("create executor failed: %w", err)
	}

	// 3. 创建 Replanner
	replanner, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: toolChatModel,
		GenInputFn: genComplexReplannerInput,
	})
	if err != nil {
		return "", fmt.Errorf("create replanner failed: %w", err)
	}

	// 4. 创建 Plan-Execute Agent
	agent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planner,
		Executor:      executor,
		Replanner:     replanner,
		MaxIterations: maxLoop,
	})
	if err != nil {
		return "", fmt.Errorf("create plan-execute agent failed: %w", err)
	}

	// 5. 运行 Agent
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent})
	query := formatComplexUserInputStatic(input.UserID, input.Query)
	iter := runner.Run(ctx, []*schema.Message{schema.UserMessage(query)})

	var lastAnswer string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
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
	return lastAnswer, nil
}

func formatComplexUserInputStatic(userID uint64, query string) string {
	return fmt.Sprintf("用户ID: %d\n查询: %s", userID, query)
}

// 以下是 Plan-Execute 的 prompt，复用现有 fallback.go 中的内容

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

var complexExecutorPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`你是一个勤奋的执行者。严格按计划执行当前步骤，并在执行后汇报结果。

## 你的工具
你可以调用以下工具完成任务：
- search_video: 搜索视频
- get_hot_videos: 获取热门视频
- get_user_info: 查询用户信息
- get_follow_list: 查询关注列表
- get_fans_list: 查询粉丝列表

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

请执行当前步骤。`),
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

func genComplexExecutorInput(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
	planJSON, _ := json.Marshal(in.Plan)
	return complexExecutorPrompt.Format(ctx, map[string]any{
		"input":          formatComplexUserInputFromMsg(in.UserInput),
		"plan":           string(planJSON),
		"executed_steps": formatComplexStepsStr(in.ExecutedSteps),
		"step":           in.Plan.FirstStep(),
	})
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

// RunComplexTaskInput 是复杂任务的输入参数。
type RunComplexTaskInput struct {
	Query  string
	UserID uint64
}

// ExecVideoAnalysisAssistant 执行视频分析任务（Plan-Execute 模式）。
// 内部复用 RunComplexTask 的逻辑。
func ExecVideoAnalysisAssistant(
	ctx context.Context,
	toolChatModel model.ToolCallingChatModel,
	tools []tool.BaseTool,
	mcpToolsInfo []*schema.ToolInfo,
	input *RunComplexTaskInput,
	maxSteps, maxLoop int,
) (string, error) {
	return RunComplexTask(ctx, toolChatModel, tools, mcpToolsInfo, input, maxSteps, maxLoop)
}
