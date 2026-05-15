package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
)

func formatCount(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func formatDuration(seconds int) string {
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func toInvokable(tools []tool.BaseTool) []tool.InvokableTool {
	result := make([]tool.InvokableTool, 0, len(tools))
	for _, t := range tools {
		if inv, ok := t.(tool.InvokableTool); ok {
			result = append(result, inv)
		}
	}
	return result
}

func findTool(tools []tool.InvokableTool, name string) tool.InvokableTool {
	for _, t := range tools {
		if info, err := t.Info(context.Background()); err == nil && info.Name == name {
			return t
		}
	}
	return nil
}

func mustMarshalJSON(v any) string {
	bs, _ := json.Marshal(v)
	return string(bs)
}

// CallGetHotVideos 调用 get_hot_videos 工具。供 video_semantic_recommend 的 fallback 路径使用。
func CallGetHotVideos(ctx context.Context, tools []tool.BaseTool, limit int) (string, error) {
	if limit <= 0 {
		limit = 5
	}
	invTools := toInvokable(tools)
	hotTool := findTool(invTools, "get_hot_videos")
	if hotTool == nil {
		return "", fmt.Errorf("get_hot_videos tool not found")
	}
	rawResult, err := hotTool.InvokableRun(ctx, mustMarshalJSON(map[string]any{"limit": limit}))
	if err != nil {
		return "", fmt.Errorf("get_hot_videos failed: %w", err)
	}
	return rawResult, nil
}

// FormatRecommendResult 格式化热门视频推荐结果，供 video_semantic_recommend 的 fallback 路径使用。
func FormatRecommendResult(raw string) string {
	var resp struct {
		Videos []VideoInfo `json:"videos"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return "暂无推荐数据。"
	}

	if len(resp.Videos) == 0 {
		return "暂无推荐内容，请稍后再试。"
	}

	return formatRecommendResultInternal(resp.Videos)
}

// formatRecommendResultInternal 内部格式化函数，共享推荐结果的展示逻辑。
func formatRecommendResultInternal(videos []VideoInfo) string {
	var sb strings.Builder
	sb.WriteString("🔥 为你推荐以下内容：\n\n")
	for i, v := range videos {
		sb.WriteString(fmt.Sprintf("%d. 《%s》\n", i+1, v.Title))
		sb.WriteString(fmt.Sprintf("   %s | ▶️ %s | 👍 %s\n", v.AuthorName, formatCount(v.PlayCount), formatCount(v.LikeCount)))
		if v.Duration > 0 {
			sb.WriteString(fmt.Sprintf("   ⏱️ %s\n", formatDuration(v.Duration)))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("💡 输入序号查看详情，或告诉我其他需求。")
	return sb.String()
}
