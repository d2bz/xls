package router

import (
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

const (
	MinConfidence = 0.65
)

// buildSupervisorPrompt 构建 Supervisor 的 LLM prompt。
// mcpToolsInfo 为 nil 时行为与之前一致。
func buildSupervisorPrompt(mcpToolsInfo []*schema.ToolInfo) string {
	mcpSection := ""
	if len(mcpToolsInfo) > 0 {
		mcpSection += "\n\n### MCP 视频分析工具（通过 complex_analysis 意图调用）\n"
		mcpSection += "当用户请求以下类型的分析任务时，intent 使用 complex_analysis，complexity 为 complex：\n"
		for _, t := range mcpToolsInfo {
			mcpSection += fmt.Sprintf("- %s: %s\n", t.Name, t.Desc)
		}
		mcpSection += "触发示例：" + buildMCPToolsExamples(mcpToolsInfo) + "\n"
		mcpSection += "注意：这类任务需要走复杂任务节点，由 LLM 通过 Plan-Execute 模式调用 MCP 工具。"
	}

	return `你是一个短视频平台助手，专门负责理解用户查询的意图。

## 支持的意图类型

### video_search
用户想搜索/查找特定主题的视频。
示例："帮我搜索搞笑视频"、"找一个关于美食的"、"有没有讲编程的"、"搜下原神相关视频"

### video_analysis
用户想分析/统计视频的热度、播放量、趋势，或找出最火/最热的视频。
包括但不限于：
- "我最火的视频是哪个"、"最近哪个视频播放量最高"
- "分析下我的视频趋势"、"哪个视频点赞最多"
- "帮我分析这个视频的画面质量"（需要调用 MCP 工具）
- "对比我最近发的几个视频的数据"（需要多步）
- "为什么我的视频播放量下降了"（需要多步分析）
- "这个视频的配乐是什么"、"视频里有没有文字"
示例："我最火的视频是哪个"、"最近哪个视频播放量最高"、"分析下我的视频趋势"、"哪个视频点赞最多"、"帮我分析这个视频的画面质量"

### user_relation
用户想了解关注/粉丝/社交关系。
示例："我关注了哪些人"、"谁关注了我"、"粉丝有多少"、"我的关注列表"

### recommend
用户想获得视频推荐。
示例："给我推荐个视频"、"推荐一个搞笑的"、"有什么好玩的"

### recommend_semantic
用户想获得带有语义描述的个性化推荐，通常包含场景、风格、情感等多维度偏好描述。
示例："推荐适合晚上学习听的日语女声视频"、"给我推荐下饭的搞笑视频"、"适合一个人看的小众治愈内容"

注意事项：
- 如果用户只是简单说"推荐一个视频"，使用 recommend 而非 recommend_semantic

### complex_analysis
复杂的分析任务，需要多步骤规划和深入研究。
示例："帮我深度分析短视频行业趋势"、"分析我的内容方向并给出建议"` + mcpSection + `

### general
日常闲聊、问候或无法归类的请求。
示例："你好"、"今天天气怎么样"、"你是谁"

## 重要规则

1. 如果用户提到"我"、"我的"，则 user_id 使用对话中提供的用户ID，不要自行生成。
2. 如果用户没有指定数量，limit 默认为 5。
3. 如果用户没有指定页码，page 默认为 1。
4. sort 可选值: hot（热度）、latest（最新）、likes（点赞数），未指定时默认 hot。
5. complexity 判断标准：
   - simple: 一次工具调用即可完成（如：搜索视频、获取用户信息）
   - medium: 需要2-3步（如：找最火的视频，需要搜索+排序）
   - complex: 需要多步或Agentic能力（如：深度趋势分析、多维度对比）、或涉及 MCP 视频分析工具

## 输出格式

严格输出以下JSON格式，不要包含任何其他内容：

{
  "tasks": [
    {
      "intent": "意图类型",
      "confidence": 置信度（0.0~1.0）,
      "slots": {
        "keyword": "搜索关键词（仅video_search需要）",
        "sort": "排序方式（仅video_analysis需要）",
        "limit": 数量,
        "page": 页码,
        "user_id": 用户ID,
        "video_id": 视频ID（如果有）",
        "author_id": "作者ID（如果有）",
        "target_uid": "目标用户ID（如果有）"
      },
      "complexity": "simple|medium|complex"
    }
  ],
  "is_multi": false,
  "reason": "判断理由（1句话）"
}

**注意**：
- 只输出一个 task，除非用户明确提出了多个独立的请求（如"先...再..."）。
- 如果置信度低于 0.65，intent 改为 fallback。
- slots 中只需要填充与该意图相关的字段，其他字段可以省略。
- 返回的 user_id 必须是对话中提供的用户ID，不要自行生成。`
}

// BuildSupervisorPromptWithoutMCP 导出函数，供 Legacy 模式使用（无 MCP 工具）。
func BuildSupervisorPromptWithoutMCP() string {
	return buildSupervisorPrompt(nil)
}

// BuildMCPToolsPrompt 生成 MCP 工具的 prompt 片段，供外部拼接。
// 返回格式化的字符串，包含所有 MCP 工具的名称和描述。
func BuildMCPToolsPrompt(mcpToolsInfo []*schema.ToolInfo) string {
	if len(mcpToolsInfo) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, t := range mcpToolsInfo {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Name, t.Desc))
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

// buildMCPToolsExamples 根据 MCP 工具生成示例描述。
func buildMCPToolsExamples(mcpToolsInfo []*schema.ToolInfo) string {
	var sb strings.Builder
	for _, t := range mcpToolsInfo {
		sb.WriteString(fmt.Sprintf("\"帮我用 %s 分析xxx\"、", t.Name))
	}
	s := sb.String()
	if len(s) > 0 {
		s = s[:len(s)-1]
	}
	return s
}

func trimCodeBlock(s string) string {
	const prefix = "```json"
	const suffix = "```"
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		if idx := findLastIndex(s, suffix); idx != -1 {
			return s[len(prefix):idx]
		}
		return s[len(prefix):]
	}
	if len(s) >= len(prefix) {
		if idx := findLastIndex(s, prefix); idx != -1 {
			rest := s[idx+len(prefix):]
			if idx := findLastIndex(rest, suffix); idx != -1 {
				return rest[:idx]
			}
			return rest
		}
	}
	return s
}

func findLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func buildHistoryContext(history []*schema.Message) string {
	if len(history) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\n## 对话历史\n")
	for i, msg := range history {
		role := "用户"
		if msg.Role == schema.Assistant {
			role = "助手"
		}
		sb.WriteString(fmt.Sprintf("%d. %s: %s\n", i+1, role, msg.Content))
	}
	return sb.String()
}
