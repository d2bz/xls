package logic

import (
	"context"
	"encoding/json"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
	"strconv"
	"xls/app/video/mq/internal/model"
	"xls/app/video/mq/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"xls/app/video/mq/internal/svc"
)

type LikeSyncMqLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLikeSyncMqLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeSyncMqLogic {
	return &LikeSyncMqLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LikeSyncMqLogic) Consume(_ context.Context, _, val string) error {
	var msg *types.CanalLikeSyncMsg
	err := json.Unmarshal([]byte(val), &msg)
	if err != nil {
		logx.Errorf("[v-like-sync]unmarshal msg: %+v err: %v", val, err)
		return err
	}
	if msg.Type != "INSERT" && msg.Type != "DELETE" {
		return nil
	}
	for _, data := range msg.Data {
		targetType, _ := strconv.Atoi(data.TargetType)
		if targetType != types.VideoLike {
			continue
		}

		targetID, _ := strconv.ParseUint(data.TargetID, 10, 64)
		db := l.svcCtx.MysqlDB

		switch msg.Type {
		case "INSERT":
			err = model.UpdateLikeCount(db, targetID, 1)
		case "DELETE":
			err = model.UpdateLikeCount(db, targetID, -1)
		}
		if err != nil {
			logx.Errorf("[v-like-sync]update like count err: %v", err)
			return err
		}
	}
	return nil
}

func Consumers(ctx context.Context, svcCtx *svc.ServiceContext) []service.Service {
	return []service.Service{
		kq.MustNewQueue(svcCtx.Config.LikeSyncKqConsumerConf, NewLikeSyncMqLogic(ctx, svcCtx)),
		kq.MustNewQueue(svcCtx.Config.VideoKqConsumerConf, NewVideoMqLogic(ctx, svcCtx)),
	}
}
