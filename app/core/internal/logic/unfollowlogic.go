package logic

import (
	"context"
	"encoding/json"
	"xls/app/core/internal/code"
	"xls/app/follow/rpc/followclient"

	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnFollowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUnFollowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnFollowLogic {
	return &UnFollowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UnFollowLogic) UnFollow(req *types.UnFollowRequest) (resp *types.UnFollowResponse, err error) {
	resp = new(types.UnFollowResponse)

	uid, err := l.ctx.Value("user_id").(json.Number).Int64()
	if err != nil {
		resp.Status = code.NoLogin
		return resp, nil
	}

	unfollow, err := l.svcCtx.FollowRpc.UnFollow(l.ctx, &followclient.UnFollowRequest{
		UserID:         uint64(uid),
		FollowedUserID: req.FollowedUserID,
	})
	if err != nil {
		l.Logger.Errorf("followRPC err: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}

	if unfollow.Error.Code != 0 {
		resp.Status.StatusCode = int(unfollow.Error.Code)
		resp.Status.StatusMsg = unfollow.Error.Message
		return resp, nil
	}

	resp.Status = code.SUCCEED

	return
}
