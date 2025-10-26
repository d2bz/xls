package logic

import (
	"context"
	"xls/app/video/rpc/video/internal/model"

	"xls/app/video/rpc/video/internal/code"
	"xls/app/video/rpc/video/internal/svc"
	"xls/app/video/rpc/video/video"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishLogic {
	return &PublishLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PublishLogic) Publish(in *video.PublishRequest) (*video.PublishResponse, error) {
	resp := new(video.PublishResponse)
	newVideo := &model.Video{
		Uid:   uint(in.Uid),
		Title: in.Title,
		Url:   in.Url,
	}
	db := l.svcCtx.MysqlDB
	err := newVideo.Insert(db)
	if err != nil {
		l.Logger.Errorf("insert video to mysql error: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}
	resp.VideoID = int32(newVideo.ID)
	resp.Error = code.SUCCEED
	return resp, nil
}
