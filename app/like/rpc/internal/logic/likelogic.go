package logic

import (
	"context"
	"encoding/json"
	"fmt"

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
	// resp := new(like.LikeResponse)
	// todo: add your logic here and delete this line
	// 判断是否点赞

	msg := &types.LikeMsg{
		UserID:     in.UserID,
		TargetID:   in.TargetID,
		TargetType: in.TargetType,
		IsLike:     1,
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

	return &like.LikeResponse{}, nil
}
