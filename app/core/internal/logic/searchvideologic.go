package logic

import (
	"context"
	"xls/app/core/internal/code"
	"xls/app/video/rpc/video/videoclient"

	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchVideoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchVideoLogic {
	return &SearchVideoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchVideoLogic) SearchVideo(req *types.SearchVideoRequest) (resp *types.SearchVideoResponse, err error) {
	resp = &types.SearchVideoResponse{}

	videos, err := l.svcCtx.VideoRpc.SearchVideo(l.ctx, &videoclient.SearchVideoRequest{
		Keyword: req.Keyword,
		Page:    req.Page,
		Size:    req.Size,
	})
	if err != nil {
		l.Logger.Errorf("search video rpc error: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}

	if videos.Error.Code != 0 {
		resp.Status.StatusCode = int(videos.Error.Code)
		resp.Status.StatusMsg = videos.Error.Message
		return resp, nil
	}

	videoResp := make([]types.VideoItem, 0, len(videos.Videos))
	for _, v := range videos.Videos {
		videoResp = append(videoResp, types.VideoItem{
			VideoID:      v.VideoID,
			AuthorID:     v.AuthorID,
			AuthorName:   v.AuthorName,
			AuthorAvatar: v.AuthorAvatar,
			Title:        v.Title,
			Url:          v.Url,
			LikeNum:      v.LikeNum,
			CommentNum:   v.CommentNum,
			CreatedAt:    v.CreatedAt,
		})
	}

	resp.Videos = videoResp
	resp.Status = code.SUCCEED

	return resp, nil
}
