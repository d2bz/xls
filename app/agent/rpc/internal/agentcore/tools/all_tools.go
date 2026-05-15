package tools

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"xls/app/agent/rpc/internal/agentcore/milvus"
	"xls/app/follow/rpc/followclient"
	"xls/app/like/rpc/likeclient"
	"xls/app/user/rpc/userclient"
	"xls/app/video/rpc/video/videoclient"
	"xls/pkg/embedding"
)

type (
	GetVideosByDimensionsInput struct {
		Dims  []DimInput `json:"dims" jsonschema:"required" jsonschema_description:"维度列表，每个维度包含名称、关键词列表和权重"`
		Limit int        `json:"limit" jsonschema:"description:"返回数量，默认10"`
		Page  int        `json:"page" jsonschema:"description:"页码，从1开始"`
	}

	DimInput struct {
		Name   string   `json:"name" jsonschema:"required,description:维度名称，如'学习'、'日语'"`
		Tags   []string `json:"tags" jsonschema:"required,description:该维度的关键词列表"`
		Weight int      `json:"weight" jsonschema:"description:权重，越小越重要"`
	}

	SearchVideoInput struct {
		Keyword  string `json:"keyword" jsonschema:"required" jsonschema_description:"搜索关键词"`
		Page     int    `json:"page" jsonschema_description:"页码，从1开始"`
		PageSize int    `json:"page_size" jsonschema_description:"每页数量"`
	}

	GetVideoInput struct {
		Limit int `json:"limit" jsonschema_description:"返回数量限制，默认10"`
	}

	UserInfoInput struct {
		UserID uint64 `json:"user_id" jsonschema:"required" jsonschema_description:"用户ID"`
	}

	FollowListInput struct {
		UserID uint64 `json:"user_id" jsonschema:"required" jsonschema_description:"用户ID"`
		Page   int    `json:"page" jsonschema_description:"页码，从1开始"`
	}

	FansListInput struct {
		UserID uint64 `json:"user_id" jsonschema:"required" jsonschema_description:"用户ID"`
		Page   int    `json:"page" jsonschema_description:"页码，从1开始"`
	}
)

func NewTools(
	videoRpc videoclient.Video,
	userRpc userclient.User,
	followRpc followclient.Follow,
	likeRpc likeclient.Like,
	embedder *embedding.Embedder,
	milvusClient *milvus.Client,
) ([]tool.BaseTool, error) {
	searchVideo, err := utils.InferTool(
		"search_video",
		"搜索视频列表，根据关键词返回匹配的视频信息",
		func(ctx context.Context, input *SearchVideoInput) (string, error) {
			page := input.Page
			if page <= 0 {
				page = 1
			}
			size := input.PageSize
			if size <= 0 {
				size = 10
			}
			resp, err := videoRpc.SearchVideo(ctx, &videoclient.SearchVideoRequest{
				Keyword: input.Keyword,
				Page:    int64(page),
				Size:    int64(size),
			})
			if err != nil {
				return "", err
			}
			bs, err := json.Marshal(resp)
			if err != nil {
				return "", err
			}
			return string(bs), nil
		},
	)
	if err != nil {
		return nil, err
	}

	hotVideos, err := utils.InferTool(
		"get_hot_videos",
		"获取热门视频列表，展示当前最受欢迎的视频",
		func(ctx context.Context, input *GetVideoInput) (string, error) {
			limit := input.Limit
			if limit <= 0 {
				limit = 10
			}
			idResp, err := likeRpc.HotVideoIDList(ctx, &likeclient.HotVideoIDListRequest{})
			if err != nil {
				return "", err
			}
			ids := idResp.VideoIDs
			if len(ids) > limit {
				ids = ids[:limit]
			}
			if len(ids) == 0 {
				return "[]", nil
			}
			resp, err := videoRpc.GetVideoList(ctx, &videoclient.GetVideoListRequest{
				VideoIDs: ids,
			})
			if err != nil {
				return "", err
			}
			bs, err := json.Marshal(resp)
			if err != nil {
				return "", err
			}
			return string(bs), nil
		},
	)
	if err != nil {
		return nil, err
	}

	userInfo, err := utils.InferTool(
		"get_user_info",
		"获取指定用户的基本信息，包括昵称、头像、粉丝数等",
		func(ctx context.Context, input *UserInfoInput) (string, error) {
			resp, err := userRpc.UserInfo(ctx, &userclient.UserInfoRequest{
				UserID: input.UserID,
			})
			if err != nil {
				return "", err
			}
			bs, err := json.Marshal(resp)
			if err != nil {
				return "", err
			}
			return string(bs), nil
		},
	)
	if err != nil {
		return nil, err
	}

	followList, err := utils.InferTool(
		"get_follow_list",
		"获取指定用户的关注列表",
		func(ctx context.Context, input *FollowListInput) (string, error) {
			page := input.Page
			if page <= 0 {
				page = 1
			}
			resp, err := followRpc.FollowList(ctx, &followclient.FollowListRequest{
				UserID:   input.UserID,
				Cursor:   0,
				PageSize: int64(page) * 10,
			})
			if err != nil {
				return "", err
			}
			bs, err := json.Marshal(resp)
			if err != nil {
				return "", err
			}
			return string(bs), nil
		},
	)
	if err != nil {
		return nil, err
	}

	fansList, err := utils.InferTool(
		"get_fans_list",
		"获取指定用户的粉丝列表",
		func(ctx context.Context, input *FansListInput) (string, error) {
			page := input.Page
			if page <= 0 {
				page = 1
			}
			resp, err := followRpc.FansList(ctx, &followclient.FansListRequest{
				UserID:   input.UserID,
				Cursor:   0,
				PageSize: int64(page) * 10,
			})
			if err != nil {
				return "", err
			}
			bs, err := json.Marshal(resp)
			if err != nil {
				return "", err
			}
			return string(bs), nil
		},
	)
	if err != nil {
		return nil, err
	}

	getVideosByDimensions, err := utils.InferTool(
		"get_videos_by_dimensions",
		"根据多个语义维度获取视频列表，每个维度通过关键词列表表示语义方向。用于语义推荐场景",
		func(ctx context.Context, input *GetVideosByDimensionsInput) (string, error) {
			page := input.Page
			if page <= 0 {
				page = 1
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 10
			}

			dims := make([]*videoclient.Dimension, 0, len(input.Dims))
			for _, d := range input.Dims {
				dims = append(dims, &videoclient.Dimension{
					Name:   d.Name,
					Tags:   d.Tags,
					Weight: int32(d.Weight),
				})
			}
			resp, err := videoRpc.GetVideosByDimensions(ctx, &videoclient.GetVideosByDimensionsRequest{
				Dimensions: dims,
				Limit:      int64(limit),
				Page:       int64(page),
			})
			if err != nil {
				return "", err
			}
			bs, err := json.Marshal(resp)
			if err != nil {
				return "", err
			}
			return string(bs), nil
		},
	)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{searchVideo, hotVideos, userInfo, followList, fansList, getVideosByDimensions}, nil
}
