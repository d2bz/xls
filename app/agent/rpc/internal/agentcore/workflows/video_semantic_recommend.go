package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
	"xls/app/agent/rpc/internal/agentcore/milvus"
	"xls/pkg/embedding"
)

// SemanticRecommendDeps 是构造 Workflow 时的外部依赖（应用级单例）。
type SemanticRecommendDeps struct {
	Tools            []tool.BaseTool
	Embedder         *embedding.Embedder
	Milvus           *milvus.Client
	MilvusSearchTopK int
	ChatModel        model.ChatModel
}

// SemanticWorkflowInput 是 Workflow 的输入类型。
type SemanticWorkflowInput struct {
	Query  string
	Limit  int
	Dims   []*SemanticDim
	UserID uint64
}

// SemanticWorkflowOutput 是 Workflow 的输出类型。
type SemanticWorkflowOutput struct {
	Answer string
}

// semanticWorkflowState 是 Workflow 内部的局部状态。
type semanticWorkflowState struct {
	Dims         []*SemanticDim
	ExpandedDims []*SemanticDim
	NeedFallback bool
	Limit        int
}

// BuildVideoSemanticRecommendWorkflow 创建并编译语义推荐 Workflow。
//
// Workflow 拓扑:
//
//	  START
//	    └─→ extract_dims ──→ expand_dims ──→ Branch{valid?}
//	                                       ├─→ query_videos ──→ END
//	                                       └─→ fallback_hot   ──→ END
//
// 各节点说明:
//   - extract_dims : 优先使用 Slots.Dims，否则调用 LLM 提取；结果写入 State
//   - expand_dims   : 并发调用 embed+Milvus 扩展每个维度的 tags；结果写入 State
//   - query_videos  : 调用 get_videos_by_dimensions 工具，直接返回格式化结果
//   - fallback_hot  : 降级为热榜推荐（作为 Workflow 内部节点，不再跨 Workflow 函数调用）
func BuildVideoSemanticRecommendWorkflow(
	ctx context.Context,
	deps SemanticRecommendDeps,
) (compose.Runnable[*SemanticWorkflowInput, *SemanticWorkflowOutput], error) {
	// ---------- 节点定义 ----------

	// extract_dims: 从 Slots.Dims 读取，兜底调用 LLM，结果写入 State
	extractDimsNode := compose.InvokableLambda(
		func(ctx context.Context, input *SemanticWorkflowInput) (string, error) {
			if err := compose.ProcessState[*semanticWorkflowState](ctx, func(ctx context.Context, state *semanticWorkflowState) error {
				state.Limit = input.Limit
				if state.Limit <= 0 {
					state.Limit = 10
				}
				return nil
			}); err != nil {
				return "", err
			}

			var dims []*SemanticDim
			if len(input.Dims) > 0 {
				dims = input.Dims
			} else if deps.ChatModel != nil {
				var err error
				dims, err = extractDimsByLLM(ctx, deps.ChatModel, input.Query)
				if err != nil {
					logx.Errorf("[semantic] extract dims by LLM failed: %v", err)
					if writeErr := compose.ProcessState[*semanticWorkflowState](ctx, func(_ context.Context, s *semanticWorkflowState) error {
						s.NeedFallback = true
						return nil
					}); writeErr != nil {
						return "", writeErr
					}
					return "", nil
				}
			}

			if len(dims) == 0 {
				if writeErr := compose.ProcessState[*semanticWorkflowState](ctx, func(_ context.Context, s *semanticWorkflowState) error {
					s.NeedFallback = true
					return nil
				}); writeErr != nil {
					return "", writeErr
				}
				return "", nil
			}

			if writeErr := compose.ProcessState[*semanticWorkflowState](ctx, func(_ context.Context, s *semanticWorkflowState) error {
				s.Dims = dims
				return nil
			}); writeErr != nil {
				return "", writeErr
			}
			return "", nil
		},
	)

	// expand_dims: 并发扩展每个维度的 tags（embed + Milvus 搜索），结果写入 State
	expandDimsNode := compose.InvokableLambda(
		func(ctx context.Context, _ string) (string, error) {
			var dims []*SemanticDim
			if readErr := compose.ProcessState[*semanticWorkflowState](ctx, func(_ context.Context, s *semanticWorkflowState) error {
				dims = s.Dims
				return nil
			}); readErr != nil {
				return "", readErr
			}

			if len(dims) == 0 {
				return "", nil
			}

			expanded, err := expandDimsConcurrent(ctx, deps, dims)
			if err != nil {
				logx.Errorf("[semantic] expand dims failed: %v", err)
				if writeErr := compose.ProcessState[*semanticWorkflowState](ctx, func(_ context.Context, s *semanticWorkflowState) error {
					s.NeedFallback = true
					return nil
				}); writeErr != nil {
					return "", writeErr
				}
				return "", nil
			}

			hasValid := false
			for _, dim := range expanded {
				if len(dim.Tags) > 0 {
					hasValid = true
					break
				}
			}
			if !hasValid {
				if writeErr := compose.ProcessState[*semanticWorkflowState](ctx, func(_ context.Context, s *semanticWorkflowState) error {
					s.NeedFallback = true
					return nil
				}); writeErr != nil {
					return "", writeErr
				}
				return "", nil
			}

			if writeErr := compose.ProcessState[*semanticWorkflowState](ctx, func(_ context.Context, s *semanticWorkflowState) error {
				s.ExpandedDims = expanded
				return nil
			}); writeErr != nil {
				return "", writeErr
			}
			return "", nil
		},
	)

	// query_videos: 调用 get_videos_by_dimensions 工具，直接返回最终结果
	queryVideosNode := compose.InvokableLambda(
		func(ctx context.Context, _ string) (*SemanticWorkflowOutput, error) {
			var expandedDims []*SemanticDim
			var limit int
			if readErr := compose.ProcessState[*semanticWorkflowState](ctx, func(_ context.Context, s *semanticWorkflowState) error {
				expandedDims = s.ExpandedDims
				limit = s.Limit
				return nil
			}); readErr != nil {
				return &SemanticWorkflowOutput{Answer: "推荐处理失败，请稍后再试。"}, nil
			}

			if len(expandedDims) == 0 {
				return &SemanticWorkflowOutput{Answer: "未能匹配到符合条件的视频，请尝试其他描述。"}, nil
			}

			rawResult, err := callGetVideosByDimensions(ctx, deps.Tools, expandedDims, limit)
			if err != nil {
				logx.Errorf("[semantic] query videos failed: %v", err)
				return &SemanticWorkflowOutput{Answer: "推荐处理失败，请稍后再试。"}, nil
			}
			return &SemanticWorkflowOutput{Answer: formatSemanticRecommendResult(rawResult)}, nil
		},
	)

	// fallback_hot: 降级为热榜推荐（Workflow 内部节点）
	fallbackHotNode := compose.InvokableLambda(
		func(ctx context.Context, _ string) (*SemanticWorkflowOutput, error) {
			var limit int
			if readErr := compose.ProcessState[*semanticWorkflowState](ctx, func(_ context.Context, s *semanticWorkflowState) error {
				limit = s.Limit
				return nil
			}); readErr != nil {
				return &SemanticWorkflowOutput{Answer: "推荐处理失败，请稍后再试。"}, nil
			}

			rawResult, err := CallGetHotVideos(ctx, deps.Tools, limit)
			if err != nil {
				logx.Errorf("[semantic] fallback hot failed: %v", err)
				return &SemanticWorkflowOutput{Answer: "推荐处理失败，请稍后再试。"}, nil
			}
			return &SemanticWorkflowOutput{Answer: FormatRecommendResult(rawResult)}, nil
		},
	)

	// ---------- 分支条件 ----------
	// 根据 State.NeedFallback 决定走 query_videos 还是 fallback_hot
	checkFallbackBranch := compose.NewGraphBranch(
		func(ctx context.Context, _ string) (string, error) {
			var needFallback bool
			if err := compose.ProcessState[*semanticWorkflowState](ctx, func(_ context.Context, s *semanticWorkflowState) error {
				needFallback = s.NeedFallback
				return nil
			}); err != nil {
				return "", err
			}
			if needFallback {
				return "fallback_hot", nil
			}
			return "query_videos", nil
		},
		map[string]bool{"query_videos": true, "fallback_hot": true},
	)

	// ---------- 构建 Workflow ----------
	wf := compose.NewWorkflow[*SemanticWorkflowInput, *SemanticWorkflowOutput](
		compose.WithGenLocalState(func(ctx context.Context) *semanticWorkflowState {
			return &semanticWorkflowState{}
		}),
	)

	// extract_dims: 从 START 接收输入
	wf.AddLambdaNode("extract_dims", extractDimsNode).
		AddInput(compose.START)

	// expand_dims: 等 extract_dims 完成
	wf.AddLambdaNode("expand_dims", expandDimsNode).
		AddInput("extract_dims")

	// Branch: expand_dims 之后根据 NeedFallback 分支
	wf.AddBranch("expand_dims", checkFallbackBranch)

	// query_videos: expand_dims 之后且 NeedFallback=false
	wf.AddLambdaNode("query_videos", queryVideosNode).
		AddInput("expand_dims")

	// fallback_hot: expand_dims 之后且 NeedFallback=true
	wf.AddLambdaNode("fallback_hot", fallbackHotNode).
		AddInput("expand_dims")

	// END: 收集 query_videos 或 fallback_hot 的结果
	wf.End().
		AddInput("query_videos").
		AddInput("fallback_hot")

	return wf.Compile(ctx)
}

// ExecVideoSemanticRecommendWorkflow 保持向后兼容：
// 构建 Workflow 并执行。如果外部传入的 Slots.Dims 为空，
// 会走 LLM 提取维度；否则直接使用传入的 dims。
func ExecVideoSemanticRecommendWorkflow(ctx context.Context, deps SemanticRecommendDeps, task *Task) (string, error) {
	wf, err := BuildVideoSemanticRecommendWorkflow(ctx, deps)
	if err != nil {
		logx.Errorf("[semantic] build workflow failed: %v", err)
		return "推荐处理失败，请稍后再试。", nil
	}

	limit := task.Slots.Limit
	if limit <= 0 {
		limit = 10
	}

	input := &SemanticWorkflowInput{
		Query:  task.Query,
		Limit:  limit,
		Dims:   task.Slots.Dims,
		UserID: task.Slots.UserID,
	}

	out, err := wf.Invoke(ctx, input)
	if err != nil {
		logx.Errorf("[semantic] workflow invoke failed: %v", err)
		return "推荐处理失败，请稍后再试。", nil
	}
	return out.Answer, nil
}

// expandDimsConcurrent 并发扩展每个维度的 tags：embed + Milvus 搜索。
func expandDimsConcurrent(ctx context.Context, deps SemanticRecommendDeps, dims []*SemanticDim) ([]*SemanticDim, error) {
	type result struct {
		dim      *SemanticDim
		expanded []string
	}

	workChan := make(chan *SemanticDim, len(dims))
	resultChan := make(chan result, len(dims))

	for i := range dims {
		workChan <- dims[i]
	}
	close(workChan)

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)

	for dim := range workChan {
		dim := dim
		sem <- struct{}{}
		go func() {
			defer func() { <-sem }()
			expanded, err := expandDimTags(ctx, deps, dim.Name, dim.Tags)
			if err != nil {
				logx.Errorf("[semantic] expand tags for dim %q failed: %v", dim.Name, err)
				resultChan <- result{dim: dim, expanded: nil}
				return
			}
			resultChan <- result{dim: dim, expanded: expanded}
		}()
	}

	var results []*SemanticDim
	for i := 0; i < len(dims); i++ {
		r := <-resultChan
		if len(r.expanded) > 0 {
			dim := &SemanticDim{
				Name:   r.dim.Name,
				Tags:   r.expanded,
				Weight: r.dim.Weight,
			}
			results = append(results, dim)
		}
	}
	return results, nil
}

// extractDimsByLLM 调用 LLM 从用户查询中提取语义维度。
func extractDimsByLLM(ctx context.Context, chatModel model.ChatModel, query string) ([]*SemanticDim, error) {
	tpl := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage(`你是一个短视频语义分析助手。从用户查询中提取多个语义维度。

## 任务
分析用户查询，提取最关键的语义维度。每个维度包含：
- name: 维度名称（如"学习"、"放松"、"日语"等核心词）
- weight: 权重（越小越重要，范围1-10，默认1）

## 规则
- 最多提取5个维度
- 只提取有明确语义倾向的维度，避免泛泛的词
- 权重：最核心的维度权重为1，依次递增
- 如果用户只是简单说"推荐一个视频"，返回空数组

## 输出格式
严格输出JSON数组，不要包含任何其他内容：
[{"name":"维度名","weight":1},{"name":"维度名","weight":2}]`),
		schema.UserMessage("{query}"),
	)

	msgs, err := tpl.Format(ctx, map[string]any{"query": query})
	if err != nil {
		return nil, fmt.Errorf("format prompt failed: %w", err)
	}

	resp, err := chatModel.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("llm generate failed: %w", err)
	}

	content := trimJSONCodeBlock(resp.Content)
	var parsed []struct {
		Name   string `json:"name"`
		Weight int    `json:"weight"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("parse dims failed: %w", err)
	}

	dims := make([]*SemanticDim, 0, len(parsed))
	for _, r := range parsed {
		if r.Name != "" {
			dims = append(dims, &SemanticDim{Name: r.Name, Weight: r.Weight})
		}
	}
	return dims, nil
}

// trimJSONCodeBlock 去除 ```json ... ``` 包裹。
func trimJSONCodeBlock(s string) string {
	s = strings.TrimSpace(s)
	const prefix = "```json"
	const suffix = "```"
	if strings.HasPrefix(s, prefix) {
		if idx := strings.LastIndex(s, suffix); idx != -1 {
			return s[len(prefix):idx]
		}
		return s[len(prefix):]
	}
	return s
}

// expandDimTags 对单个维度的关键词做 embed + Milvus 语义搜索，扩展 tag 列表。
func expandDimTags(ctx context.Context, deps SemanticRecommendDeps, dimName string, keywords []string) ([]string, error) {
	if deps.Embedder == nil || deps.Milvus == nil {
		return keywords, nil
	}
	if len(keywords) == 0 {
		return nil, nil
	}

	topK := deps.MilvusSearchTopK
	if topK <= 0 {
		topK = 10
	}

	combined := strings.Join(keywords, " ")
	vectors, err := deps.Embedder.EmbedStrings(ctx, []string{combined})
	if err != nil {
		return keywords, fmt.Errorf("embed failed: %w", err)
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return keywords, nil
	}

	matchedTags, err := deps.Milvus.SearchTags(ctx, toFloat32(vectors[0]), topK)
	if err != nil {
		return keywords, fmt.Errorf("milvus search failed: %w", err)
	}

	seen := make(map[string]struct{})
	for _, t := range keywords {
		seen[t] = struct{}{}
	}
	result := make([]string, 0, len(keywords)+len(matchedTags))
	result = append(result, keywords...)
	for _, t := range matchedTags {
		if _, ok := seen[t]; !ok {
			result = append(result, t)
			seen[t] = struct{}{}
		}
	}
	return result, nil
}

// callGetVideosByDimensions 调用 get_videos_by_dimensions RPC。
func callGetVideosByDimensions(ctx context.Context, tools []tool.BaseTool, dims []*SemanticDim, limit int) (string, error) {
	invTools := toInvokable(tools)
	dimTool := findTool(invTools, "get_videos_by_dimensions")
	if dimTool == nil {
		return "", fmt.Errorf("get_videos_by_dimensions tool not found")
	}

	toolDims := make([]struct {
		Name   string   `json:"name"`
		Tags   []string `json:"tags"`
		Weight int     `json:"weight"`
	}, 0, len(dims))
	for _, d := range dims {
		if len(d.Tags) == 0 {
			continue
		}
		toolDims = append(toolDims, struct {
			Name   string   `json:"name"`
			Tags   []string `json:"tags"`
			Weight int     `json:"weight"`
		}{
			Name:   d.Name,
			Tags:   d.Tags,
			Weight: d.Weight,
		})
	}
	if len(toolDims) == 0 {
		return "{\"videos\":[],\"total\":0}", nil
	}

	rawResult, err := dimTool.InvokableRun(ctx, mustMarshalJSON(map[string]any{
		"dims":  toolDims,
		"limit": limit,
		"page":  1,
	}))
	if err != nil {
		return "", fmt.Errorf("get_videos_by_dimensions failed: %w", err)
	}
	return rawResult, nil
}

func formatSemanticRecommendResult(raw string) string {
	var rawResp struct {
		Videos []struct {
			Title      string   `json:"title"`
			AuthorName string   `json:"author_name"`
			LikeNum    int64    `json:"like_num"`
			Duration   int      `json:"duration"`
			Tags       []string `json:"tags"`
		} `json:"videos"`
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal([]byte(raw), &rawResp); err != nil {
		return "推荐处理失败，请稍后再试。"
	}

	if len(rawResp.Videos) == 0 {
		return "未能匹配到符合条件的视频，请尝试其他描述。"
	}

	categories := []string{"为你精选", "个性化推荐", "为你推荐", "精选内容"}
	category := categories[rand.Intn(len(categories))]

	// 构建结构化 JSON 元数据（供 FillStructuredResponse 解析）
	meta := struct {
		Videos []VideoInfo `json:"videos"`
		Total  int64       `json:"total"`
	}{
		Videos: make([]VideoInfo, len(rawResp.Videos)),
		Total:  rawResp.Total,
	}
	for i, v := range rawResp.Videos {
		meta.Videos[i] = VideoInfo{
			Title:      v.Title,
			AuthorName: v.AuthorName,
			LikeCount:  v.LikeNum,
			Duration:   v.Duration,
			Tags:       v.Tags,
		}
	}
	metaJSON, _ := json.Marshal(meta)

	var sb strings.Builder
	sb.WriteString(string(metaJSON))
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("[%s]\n\n找到 %d 个匹配视频：\n\n", category, rawResp.Total))

	for i, v := range rawResp.Videos {
		sb.WriteString(fmt.Sprintf("%d. 《%s》\n", i+1, v.Title))
		sb.WriteString(fmt.Sprintf("   %s | 👍 %s", v.AuthorName, formatCount(v.LikeNum)))
		if len(v.Tags) > 0 {
			sb.WriteString(fmt.Sprintf(" | tag: %s", strings.Join(v.Tags, ",")))
		}
		sb.WriteString("\n")
		if v.Duration > 0 {
			sb.WriteString(fmt.Sprintf("   [%s]\n", formatDuration(v.Duration)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("输入序号查看详情，或告诉我其他需求。")
	return sb.String()
}

func toFloat32(src []float64) []float32 {
	f := make([]float32, len(src))
	for i, v := range src {
		f[i] = float32(v)
	}
	return f
}
