package logic

import (
	"context"
	"xls/app/video/rpc/video/internal/code"
	"xls/app/video/rpc/video/internal/model"
	"xls/app/video/rpc/video/internal/svc"
	"xls/app/video/rpc/video/video"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetVideosByTagLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetVideosByTagLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVideosByTagLogic {
	return &GetVideosByTagLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetVideosByTagLogic) GetVideosByTag(in *video.GetVideosByTagRequest) (*video.GetVideosByTagResponse, error) {
	resp := &video.GetVideosByTagResponse{}

	if in.Page <= 0 {
		in.Page = 1
	}
	if in.Size <= 0 {
		in.Size = 10
	}
	offset := int((in.Page - 1) * in.Size)
	limit := int(in.Size)

	videos, total, err := model.FindVideosByTag(l.svcCtx.MysqlDB, in.Tag, offset, limit)
	if err != nil {
		l.Logger.Errorf("[GetVideosByTag] FindVideosByTag err: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	for _, v := range videos {
		var tags []string
		for _, tag := range v.Tags {
			tags = append(tags, tag.Name)
		}
		resp.Videos = append(resp.Videos, &video.VideoItem{
			VideoID:    int32(v.ID),
			AuthorID:   int32(v.Uid),
			Title:      v.Title,
			Url:        v.Url,
			LikeNum:    int32(v.LikeNum),
			CommentNum: int32(v.CommentNum),
			Tags:       tags,
		})
	}

	resp.Total = total
	resp.Error = code.SUCCEED

	return resp, nil
}
