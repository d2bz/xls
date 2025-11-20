package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
	"time"
	"xls/app/user/mq/internal/svc"
	"xls/app/user/mq/internal/types"

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

func (l *MqLogic) Consume(_ context.Context, key, val string) error {
	var msg *types.CanalUserInfoMsg
	err := json.Unmarshal([]byte(val), &msg)
	if err != nil {
		logx.Errorf("[user-info-mq]unmarshal msg key: %v val: %+v err: %v", key, val, err)
		return err
	}

	if len(msg.Data) == 0 {
		logx.Info("[user-info-mq] no data, skip.")
		return nil
	}

	u := msg.Data[0]
	redisKey := fmt.Sprintf("user:info:%s", u.ID)

	cacheBytes, err := json.Marshal(u)
	if err != nil {
		logx.Errorf("[user-info-mq] marshal msg err: %v", err)
		return err
	}

	expireSec := int(time.Hour.Seconds())
	err = l.svcCtx.BizRedis.SetexCtx(l.ctx, redisKey, string(cacheBytes), expireSec)
	if err != nil {
		logx.Errorf("[user-info-mq] redis setex failed key: %s err: %v", redisKey, err)
		return err
	}

	logx.Infof("[user-info-mq] redis userInfo updated key: %s", redisKey)

	return nil
}

func Consumers(ctx context.Context, svcCtx *svc.ServiceContext) []service.Service {
	return []service.Service{
		kq.MustNewQueue(svcCtx.Config.KqConsumerConf, NewMqLogic(ctx, svcCtx)),
	}
}
