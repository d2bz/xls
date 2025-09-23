package logic

import (
	"context"

	"xls/app/like/mq/internal/svc"
	"xls/app/like/mq/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MqLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMqLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MqLogic {
	return &MqLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MqLogic) Mq(req *types.Request) (resp *types.Response, err error) {
	// todo: add your logic here and delete this line

	return
}
