package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"xls/app/like/rpc/internal/code"
	"xls/app/like/rpc/internal/model"

	"xls/app/like/rpc/internal/svc"
	"xls/app/like/rpc/internal/types"
	"xls/app/like/rpc/like"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

const likePrefix = "like#"

type LikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LikeLogic) Like(in *like.LikeRequest) (*like.LikeResponse, error) {
	resp := new(like.LikeResponse)

	// 判断是否点赞
	var isLike int32
	targetID, uid := strconv.FormatUint(in.TargetID, 10), strconv.FormatUint(in.UserID, 10)
	likeKey := likePrefix + targetID
	score, err := l.svcCtx.BizRedis.Zscore(likeKey, uid)
	if err != nil {
		l.Logger.Errorf("redis get score of isLike err: %v", err)
	}

	if score > 1 {
		isLike = 1
		unlike(l, likeKey, uid)
	} else {
		db := l.svcCtx.MysqlDB
		var lk model.Like
		ok, err := lk.IsLike(db, in.TargetID, in.UserID)
		if err != nil {
			l.Logger.Errorf("database get isLike err: %v", err)
			resp.Error = code.FAILED
			return resp, nil
		}
		if ok {
			isLike = 1
			unlike(l, likeKey, uid)
		} else {
			isLike = 0
			_, err = l.svcCtx.BizRedis.Zadd(likeKey, time.Now().UnixMilli(), uid)
			if err != nil {
				l.Logger.Errorf("redis zadd isLike err: %v", err)
			}
		}
	}

	// 发送kq消息
	msg := &types.LikeMsg{
		UserID:     in.UserID,
		TargetID:   in.TargetID,
		TargetType: in.TargetType,
		IsLike:     isLike,
	}

	threading.GoSafe(func() {
		data, err := json.Marshal(msg)
		if err != nil {
			l.Logger.Errorf("[like] marshal msg: %v error: %v", msg, err)
			return
		}
		key := fmt.Sprintf("%d-%d", msg.TargetType, msg.TargetID)
		err = l.svcCtx.KqPusherClient.PushWithKey(context.Background(), key, string(data))
		if err != nil {
			l.Logger.Errorf("[like] kq push data: %v error: %v", data, err)
		}
	})

	resp.Error = code.SUCCEED

	return resp, nil
}

func unlike(l *LikeLogic, likeKey, uid string) {
	_, err := l.svcCtx.BizRedis.Zrem(likeKey, uid)
	if err != nil {
		l.Logger.Errorf("redis zrem err: %v", err)
	}
}
