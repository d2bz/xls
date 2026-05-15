package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/tool"
)

func ExecVideoAnalysisWorkflow(ctx context.Context, tools []tool.BaseTool, task *Task) (string, error) {
	invTools := toInvokable(tools)
	hotTool := findTool(invTools, "get_hot_videos")
	if hotTool == nil {
		return "", fmt.Errorf("get_hot_videos tool not found")
	}

	limit := task.Slots.Limit
	if limit <= 0 {
		limit = 5
	}

	rawResult, err := hotTool.InvokableRun(ctx, mustMarshalJSON(map[string]any{"limit": limit}))
	if err != nil {
		return "", fmt.Errorf("get_hot_videos failed: %w", err)
	}

	return formatVideoAnalysisResult(rawResult, task.Slots.Sort), nil
}

func formatVideoAnalysisResult(raw string, sortField string) string {
	var resp struct {
		Videos []VideoInfo `json:"videos"`
	}
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return "暂无热度数据。"
	}

	if len(resp.Videos) == 0 {
		return "暂未获取到热度数据，请稍后再试。"
	}

	videos := resp.Videos
	if sortField == "likes" {
		sort.Slice(videos, func(i, j int) bool {
			return videos[i].LikeCount > videos[j].LikeCount
		})
	} else {
		sort.Slice(videos, func(i, j int) bool {
			return videos[i].PlayCount > videos[j].PlayCount
		})
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("以下是当前最热门的视频 TOP%d：\n\n", len(videos)))

	for i, v := range videos {
		rank := i + 1
		medal := ""
		if rank == 1 {
			medal = "第"
		} else if rank == 2 {
			medal = "第"
		} else if rank == 3 {
			medal = "第"
		}
		sb.WriteString(fmt.Sprintf("%s%d名：《%s》\n", medal, rank, v.Title))
		sb.WriteString(fmt.Sprintf("   作者: %s | 播放: %s | 点赞: %s\n", v.AuthorName, formatCount(v.PlayCount), formatCount(v.LikeCount)))
		if v.Duration > 0 {
			sb.WriteString(fmt.Sprintf("   时长: %s\n\n", formatDuration(v.Duration)))
		}
	}

	sb.WriteString("提示：点击视频封面可查看详情，或告诉我其他问题。")
	return sb.String()
}
