package logic

import (
	"context"
	"xls/app/core/internal/code"
	"xls/app/like/rpc/likeclient"
	"xls/app/video/rpc/video/videoclient"

	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type HotVideoListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewHotVideoListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HotVideoListLogic {
	return &HotVideoListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *HotVideoListLogic) HotVideoList(req *types.HotVideoListRequest) (resp *types.HotVideoListResponse, err error) {
	resp = new(types.HotVideoListResponse)

	hotVideoIDs, err := l.svcCtx.LikeRpc.HotVideoIDList(l.ctx, &likeclient.HotVideoIDListRequest{})
	if err != nil {
		l.Logger.Errorf("[HotVideoListLogic] likeRpc.HotVideoIDList error: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}

	videoList, err := l.svcCtx.VideoRpc.GetVideoList(l.ctx, &videoclient.GetVideoListRequest{
		VideoIDs: hotVideoIDs.VideoIDs,
	})
	if err != nil {
		l.Logger.Errorf("GetVideoList rpc error:%v", err)
		resp.Status = code.FAILED
		return resp, nil
	}
	if videoList.Error.Code != 0 {
		resp.Status.StatusCode = int(videoList.Error.Code)
		resp.Status.StatusMsg = videoList.Error.Message
		return resp, nil
	}

	videoLists := make([]types.VideoItem, 0, len(videoList.HotVideoList))
	for _, v := range videoList.HotVideoList {
		videoLists = append(videoLists, types.VideoItem{
			VideoID:    v.VideoID,
			AuthorID:   v.AuthorID,
			Title:      v.Title,
			Url:        v.Url,
			LikeNum:    v.LikeNum,
			CommentNum: v.CommentNum,
		})
	}

	resp.Status = code.SUCCEED
	resp.VideoList = videoLists

	return resp, nil
}
