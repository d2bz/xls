package logic

import (
	"context"
	"encoding/json"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"strconv"
	"xls/app/comment/mq/internal/model"
	"xls/app/comment/mq/internal/svc"
	"xls/app/comment/mq/internal/types"
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

func (l *MqLogic) Consume(_ context.Context, _, val string) error {
	var msg *types.CanalMsg
	err := json.Unmarshal([]byte(val), &msg)

	if err != nil {
		logx.Errorf("[c-like-sync]unmarshal msg: %+v err: %v", val, err)
		return err
	}
	if msg.Type != "INSERT" && msg.Type != "DELETE" {
		return nil
	}
	for _, data := range msg.Data {
		targetType, _ := strconv.Atoi(data.TargetType)
		if targetType != 1 {
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
		kq.MustNewQueue(svcCtx.Config.KqConsumerConf, NewMqLogic(ctx, svcCtx)),
	}
}
