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
	zsetKey := LikeZsetKey(targetID, in.TargetType)
	statusKey := LikeStatusKey(uid, in.TargetType, targetID)

	// 优先使用状态缓存判断
	status, err := l.svcCtx.BizRedis.GetCtx(l.ctx, statusKey)
	if err != nil {
		l.Logger.Errorf("redis get like status err: %v", err)
	}

	if status == "1" {
		isLike = 1
		l.unlike(statusKey, zsetKey, uid)
	} else {
		// 状态缓存未命中，再查ZSet确认
		score, err := l.svcCtx.BizRedis.Zscore(zsetKey, uid)
		if err != nil {
			l.Logger.Errorf("redis zscore err: %v", err)
		}

		if score > 1 {
			isLike = 1
			l.unlike(statusKey, zsetKey, uid)
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
				l.unlike(statusKey, zsetKey, uid)
			} else {
				isLike = 0
				_, err = l.svcCtx.BizRedis.Zadd(zsetKey, time.Now().UnixMilli(), uid)
				if err != nil {
					l.Logger.Errorf("redis zadd isLike err: %v", err)
				}
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

func (l *LikeLogic) unlike(statusKey, zsetKey, uid string) {
	_, err := l.svcCtx.BizRedis.DelCtx(l.ctx, statusKey)
	if err != nil {
		l.Logger.Errorf("redis del err: %v", err)
	}
	_, err = l.svcCtx.BizRedis.ZremCtx(l.ctx, zsetKey, uid)
	if err != nil {
		l.Logger.Errorf("redis zrem err: %v", err)
	}
}

func LikeZsetKey(targetID string, targetType int32) string {
	if targetType == types.VideoLike {
		return fmt.Sprintf("like#zset#video#%s", targetID)
	}
	return fmt.Sprintf("like#zset#comment#%s", targetID)
}

func LikeStatusKey(uid string, targetType int32, targetID string) string {
	if targetType == types.VideoLike {
		return fmt.Sprintf("like:status:%s:video:%s", uid, targetID)
	}
	return fmt.Sprintf("like:status:%s:comment:%s", uid, targetID)
}
