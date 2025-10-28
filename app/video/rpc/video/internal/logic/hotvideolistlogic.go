package logic

import (
	"context"
	"github.com/zeromicro/go-zero/core/mr"
	"strconv"
	"xls/app/like/rpc/likeclient"
	"xls/app/video/rpc/video/internal/code"
	"xls/app/video/rpc/video/internal/model"

	"xls/app/video/rpc/video/internal/svc"
	"xls/app/video/rpc/video/video"

	"github.com/zeromicro/go-zero/core/logx"
)

type HotVideoListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHotVideoListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HotVideoListLogic {
	return &HotVideoListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *HotVideoListLogic) HotVideoList(in *video.HotVideoListRequest) (*video.HotVideoListResponse, error) {
	resp := new(video.HotVideoListResponse)

	videoIDList, err := l.svcCtx.LikeRPC.HotVideoIDList(l.ctx, &likeclient.HotVideoIDListRequest{})
	if err != nil {
		l.Logger.Errorf("[HotVideoList] likeRPC.HotVideoIDList error: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	videos, err := l.videoListByIDs(videoIDList.VideoIDs)
	if err != nil {
		l.Logger.Errorf("[HotVideoList] videoListByIDs error: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	var videoItems []*video.VideoItem
	for _, v := range videos {
		videoItems = append(videoItems, &video.VideoItem{
			VideoID:    int32(v.ID),
			Title:      v.Title,
			Url:        v.Url,
			LikeNum:    int32(v.LikeNum),
			CommentNum: int32(v.CommentNum),
		})
	}

	resp.HotVideoList = videoItems
	resp.Error = code.SUCCEED

	return resp, nil
}

func (l *HotVideoListLogic) videoListByIDs(videoIDList []string) ([]*model.Video, error) {
	videos, err := mr.MapReduce[string, *model.Video, []*model.Video](func(source chan<- string) {
		for _, videoID := range videoIDList {
			source <- videoID
		}
	}, func(idStr string, writer mr.Writer[*model.Video], cancel func(error)) {
		videoID, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			cancel(err)
			return
		}
		v, err := model.FindVideoByID(l.svcCtx.MysqlDB, uint(videoID))
		if err != nil {
			cancel(err)
			return
		}
		writer.Write(v)
	}, func(pipe <-chan *model.Video, writer mr.Writer[[]*model.Video], cancel func(error)) {
		var videos []*model.Video
		for v := range pipe {
			videos = append(videos, v)
		}
		writer.Write(videos)
	})
	if err != nil {
		return nil, err
	}
	return videos, nil
}
