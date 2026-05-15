package logic

import (
	"context"
	"strconv"

	"xls/app/video/rpc/video/internal/code"
	"xls/app/video/rpc/video/internal/model"

	"xls/app/video/rpc/video/internal/svc"
	"xls/app/video/rpc/video/video"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetVideoListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetVideoListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVideoListLogic {
	return &GetVideoListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetVideoListLogic) GetVideoList(in *video.GetVideoListRequest) (*video.GetVideoListResponse, error) {
	resp := new(video.GetVideoListResponse)

	videos, err := l.videoListByIDs(in.VideoIDs)
	if err != nil {
		l.Logger.Errorf("[GetVideoList] videoListByIDs error: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	var videoItems []*video.VideoItem
	for _, v := range videos {
		var tags []string

		for _, tag := range v.Tags {
			tags = append(tags, tag.Name)
		}

		videoItems = append(videoItems, &video.VideoItem{
			VideoID:    int32(v.ID),
			Title:      v.Title,
			Url:        v.Url,
			LikeNum:    int32(v.LikeNum),
			CommentNum: int32(v.CommentNum),
			Tags:       tags,
		})
	}

	resp.HotVideoList = videoItems
	resp.Error = code.SUCCEED

	return resp, nil
}

func (l *GetVideoListLogic) videoListByIDs(videoIDList []string) ([]*model.Video, error) {
	// // MapReduce 版本：每个 ID 单独查（不推荐，N 次 DB 查询）
	// videos, err := mr.MapReduce[string, *model.Video, []*model.Video](func(source chan<- string) {
	// 	for _, videoID := range videoIDList {
	// 		source <- videoID
	// 	}
	// }, func(idStr string, writer mr.Writer[*model.Video], cancel func(error)) {
	// 	videoID, err := strconv.ParseUint(idStr, 10, 64)
	// 	if err != nil {
	// 		cancel(err)
	// 		return
	// 	}
	// 	v, err := model.FindVideoByID(l.svcCtx.MysqlDB, uint(videoID))
	// 	if err != nil {
	// 		cancel(err)
	// 		return
	// 	}
	// 	writer.Write(v)
	// }, func(pipe <-chan *model.Video, writer mr.Writer[[]*model.Video], cancel func(error)) {
	// 	var videos []*model.Video
	// 	for v := range pipe {
	// 		videos = append(videos, v)
	// 	}
	// 	writer.Write(videos)
	// })
	// if err != nil {
	// 	return nil, err
	// }
	// return videos, nil

	// 批量查询：单次 IN 查询，高效
	ids := make([]uint, 0, len(videoIDList))
	for _, idStr := range videoIDList {
		videoID, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, uint(videoID))
	}
	return model.FindVideosByIDs(l.svcCtx.MysqlDB, ids)
}
