package logic

import (
	"context"
	"encoding/json"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/threading"
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
		logx.Errorf("unmarshal msg key: %v val: %+v err: %v", key, val, err)
		return nil
	}
	if msg.TargetType != 1 {
		return nil
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
			l.Logger.Errorf("insert likeMsg: %+v err: %v", msg, err)
			retry(l, key, val)
			return nil
		}
	} else {
		err = lk.RemoveLike(db)
		if err != nil {
			l.Logger.Errorf("remove likeMsg: %+v err: %v", msg, err)
			retry(l, key, val)
			return nil
		}
	}
	return nil
}

func Consumers(ctx context.Context, svcCtx *svc.ServiceContext) []service.Service {
	return []service.Service{
		kq.MustNewQueue(svcCtx.Config.KqConsumerConf, NewMqLogic(ctx, svcCtx)),
	}
}

// 失败的消息写回原队列重试，有死循环风险，优化方案为另开一个重试队列和死信队列
func retry(l *MqLogic, key, val string) {
	threading.GoSafe(func() {
		err := l.svcCtx.KqPusherClient.PushWithKey(context.Background(), key, val)
		if err != nil {
			l.Logger.Errorf("[like] kq push data: %v error: %v", val, err)
		}
	})
}
