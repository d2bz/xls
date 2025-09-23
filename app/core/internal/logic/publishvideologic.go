package logic

import (
	"context"
	"encoding/json"

	"xls/app/core/internal/code"
	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"
	"xls/app/video/rpc/video/video"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishVideoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPublishVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishVideoLogic {
	return &PublishVideoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PublishVideoLogic) PublishVideo(req *types.PublishVideoRequest) (resp *types.PublishVideoResponse, err error) {
	resp = new(types.PublishVideoResponse)

	uid, err := l.ctx.Value("userid").(json.Number).Int64()
	if err != nil {
		resp.Status = code.NoLogin
		return resp, nil
	}

	video, err := l.svcCtx.VideoRpc.Publish(l.ctx, &video.PublishRequest{
		Uid:   int32(uid),
		Title: req.Title,
		Url:   req.Url,
	})
	if err != nil {
		l.Logger.Errorf("Publish rpc failed: %v", err)
		return resp, nil
	}
	if video.Error.Code != 0 {
		resp.Status.StatusCode = int(video.Error.Code)
		resp.Status.StatusMsg = video.Error.Message
		return resp, nil
	}
	resp.VideoID = int(video.VideoID)
	resp.Status = code.SUCCEED

	return resp, nil
}
