package logic

import (
	"context"
	"encoding/json"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
	"xls/app/like/mq/internal/model"
	"xls/app/like/mq/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"xls/app/like/mq/internal/svc"
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

func (l *MqLogic) Consume(_ context.Context, key, val string) error {
	var msg *types.LikeMsg
	err := json.Unmarshal([]byte(val), &msg)
	if err != nil {
		logx.Errorf("[like-mq]unmarshal msg key: %v val: %+v err: %v", key, val, err)
		return err
	}
	// 写入数据库
	db := l.svcCtx.MysqlDB
	lk := &model.Like{
		UserID:     msg.UserID,
		TargetID:   msg.TargetID,
		TargetType: msg.TargetType,
	}
	if msg.IsLike == 0 {
		err = lk.InsertLike(db)
		if err != nil {
			logx.Errorf("[like-mq]insert likeMsg: %+v err: %v", msg, err)
			return err
		}
	} else {
		err = lk.RemoveLike(db)
		if err != nil {
			logx.Errorf("[like-mq]remove likeMsg: %+v err: %v", msg, err)
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
