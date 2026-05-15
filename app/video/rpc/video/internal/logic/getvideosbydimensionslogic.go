package logic

import (
	"context"

	"xls/app/video/rpc/video/internal/model"
	"xls/app/video/rpc/video/internal/svc"
	"xls/app/video/rpc/video/video"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetVideosByDimensionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetVideosByDimensionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetVideosByDimensionsLogic {
	return &GetVideosByDimensionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetVideosByDimensionsLogic) GetVideosByDimensions(in *video.GetVideosByDimensionsRequest) (*video.GetVideosByDimensionsResponse, error) {
	dimensions := in.GetDimensions()
	if len(dimensions) == 0 {
		return &video.GetVideosByDimensionsResponse{
			Error:  &video.Error{Code: 0, Message: ""},
			Videos: []*video.VideoItem{},
			Total:  0,
		}, nil
	}

	limit := in.GetLimit()
	if limit <= 0 {
		limit = 10
	}
	page := in.GetPage()
	if page <= 0 {
		page = 1
	}
	offset := int((page - 1) * limit)

	dimInputs := make([]model.DimensionInput, 0, len(dimensions))
	for _, d := range dimensions {
		if len(d.GetTags()) == 0 {
			continue
		}
		dimInputs = append(dimInputs, model.DimensionInput{
			Name:   d.GetName(),
			Tags:   d.GetTags(),
			Weight: int(d.GetWeight()),
		})
	}

	videos, total, err := model.FindVideosByDimensions(l.svcCtx.MysqlDB, dimInputs, int(limit), offset)
	if err != nil {
		l.Errorf("FindVideosByDimensions failed: %v", err)
		return &video.GetVideosByDimensionsResponse{
			Error:  &video.Error{Code: 1, Message: "query failed"},
			Videos: []*video.VideoItem{},
			Total:  0,
		}, nil
	}

	items := make([]*video.VideoItem, 0, len(videos))
	for _, v := range videos {
		tags := make([]string, 0, len(v.Tags))
		for _, t := range v.Tags {
			tags = append(tags, t.Name)
		}
		items = append(items, &video.VideoItem{
			VideoID:    int32(v.ID),
			AuthorID:   int32(v.Uid),
			Title:      v.Title,
			Url:        v.Url,
			LikeNum:    int32(v.LikeNum),
			CommentNum: int32(v.CommentNum),
			Tags:       tags,
		})
	}

	return &video.GetVideosByDimensionsResponse{
		Error:  &video.Error{Code: 0, Message: ""},
		Videos: items,
		Total:  total,
	}, nil
}
