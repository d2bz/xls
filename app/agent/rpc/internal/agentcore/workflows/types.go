package workflows

import (
	"github.com/cloudwego/eino/schema"
	"xls/app/agent/rpc/agent"
)

// VideoItem 是工作流返回视频列表时的单个视频项。
// 字段与 proto agent.VideoItem 对齐。
type VideoItem = agent.VideoItem

// WorkflowResult 是所有工作流的统一返回类型。
//
// Intent 决定了 ResultType 的填充方式：
//   - recommend / recommend_semantic / video_search → ResultType=VideoList, Videos 填充
//   - video_analysis / complex_analysis / general   → ResultType=Text, Text 填充
//   - user_relation                                 → ResultType=Text, Text 填充
//   - fallback                                      → ResultType=Text, Text 填充
type WorkflowResult struct {
	ResultType ResultType   `json:"resultType"`
	Text       string       `json:"text,omitempty"`       // 友好的 AI 回复文本
	Videos     []*VideoItem `json:"videos,omitempty"`     // 视频列表（推荐/搜索类）
	Total      int64        `json:"total,omitempty"`     // 总数
}

// ResultType 标识工作流返回结果的类型，供调用方（Router / Agent）决定展示方式。
type ResultType string

const (
	ResultTypeVideoList ResultType = "video_list" // 包含视频列表，前端渲染卡片
	ResultTypeText      ResultType = "text"       // 纯文本回复（分析/闲聊）
	ResultTypeError     ResultType = "error"      // 错误
)

// VideoInfoToItems 将 []VideoInfo（工具 JSON 解析结果）转换为 []*VideoItem（proto 类型）。
func VideoInfoToItems(infos []VideoInfo) []*VideoItem {
	if len(infos) == 0 {
		return nil
	}
	items := make([]*VideoItem, 0, len(infos))
	for _, info := range infos {
		items = append(items, &VideoItem{
			VideoID:     int32(info.ID),
			AuthorID:    int32(info.AuthorID),
			AuthorName:  info.AuthorName,
			Title:       info.Title,
			LikeNum:     int32(info.LikeCount),
			CreatedAt:   info.CreateTime,
			Tags:        info.Tags,
		})
	}
	return items
}

type Intent string

const (
	IntentVideoSearch        Intent = "video_search"
	IntentVideoAnalysis     Intent = "video_analysis"
	IntentUserRelation      Intent = "user_relation"
	IntentRecommend         Intent = "recommend"
	IntentRecommendSemantic Intent = "recommend_semantic"
	IntentComplexAnalysis   Intent = "complex_analysis"
	IntentGeneral          Intent = "general"
	IntentFallback          Intent = "fallback"
	IntentSimple           Intent = "simple"
)

type SemanticDim struct {
	Name   string   `json:"name,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	Weight int      `json:"weight,omitempty"`
}

type TaskSlot struct {
	Keyword   string         `json:"keyword,omitempty"`
	Sort      string         `json:"sort,omitempty"`
	Limit     int            `json:"limit,omitempty"`
	Page      int            `json:"page,omitempty"`
	UserID    uint64         `json:"user_id,omitempty"`
	VideoID   uint64         `json:"video_id,omitempty"`
	AuthorID  uint64         `json:"author_id,omitempty"`
	TargetUID uint64         `json:"target_uid,omitempty"`
	Dims      []*SemanticDim `json:"dims,omitempty"`
}

type Complexity string

const (
	ComplexitySimple   Complexity = "simple"
	ComplexityMedium  Complexity = "medium"
	ComplexityComplex Complexity = "complex"
)

type Task struct {
	Query      string      `json:"query,omitempty"`
	Intent     Intent      `json:"intent"`
	Slots      *TaskSlot   `json:"slots"`
	Complexity Complexity  `json:"complexity"`
	Confidence float64     `json:"confidence"`
}

// VideoInfo 视频基础信息结构，供格式化函数使用。
type VideoInfo struct {
	ID         uint64   `json:"id"`
	Title      string   `json:"title"`
	CoverURL   string   `json:"cover_url"`
	AuthorID   uint64   `json:"author_id"`
	AuthorName string   `json:"author_name"`
	PlayCount  int64    `json:"play_count"`
	LikeCount  int64    `json:"like_count"`
	Duration   int      `json:"duration"`
	CreateTime string   `json:"create_time"`
	Tags       []string `json:"tags,omitempty"`
}

// RunSimpleTaskInput 是 RunSimpleTask 的输入参数。
// 包含完整的 Task 信息，其中 Slots 包含所有精确参数（video_id、keyword、page 等）。
type RunSimpleTaskInput struct {
	Task *Task
}

// RunComplexTaskInputV2 是 RunComplexTaskV2 的输入参数。
// 包含完整的 Task 信息和 MCP 工具信息。
type RunComplexTaskInputV2 struct {
	Task         *Task
	MCPToolsInfo []*schema.ToolInfo
}

// RunComplexTaskInput 是 RunComplexTask 的输入参数（遗留类型，仅保留以兼容外部调用者）。
// 新代码应使用 RunComplexTaskInputV2。
type RunComplexTaskInput struct {
	Query  string
	UserID uint64
}
