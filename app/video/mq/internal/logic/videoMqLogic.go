package logic

import (
	"context"
	"encoding/json"
	"xls/app/video/mq/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"xls/app/video/mq/internal/svc"
)

type VideoMqLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVideoMqLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VideoMqLogic {
	return &VideoMqLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VideoMqLogic) Consume(_ context.Context, _, val string) error {
	var msg *types.CanalVideoMsg
	err := json.Unmarshal([]byte(val), &msg)
	if err != nil {
		logx.Errorf("[video-mq] unmarshal msg: %+v err: %v", val, err)
		return err
	}

	return nil
}
