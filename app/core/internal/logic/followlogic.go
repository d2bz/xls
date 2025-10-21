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

type FollowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFollowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowLogic {
	return &FollowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FollowLogic) Follow(req *types.FollowRequest) (resp *types.FollowResponse, err error) {
	resp = new(types.FollowResponse)

	uid, err := l.ctx.Value("user_id").(json.Number).Int64()
	if err != nil {
		resp.Status = code.NoLogin
		return resp, nil
	}

	follow, err := l.svcCtx.FollowRpc.Follow(l.ctx, &followclient.FollowRequest{
		UserID:         uint64(uid),
		FollowedUserID: req.FollowedUserID,
	})
	if err != nil {
		l.Logger.Errorf("followRPC err: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}

	if follow.Error.Code != 0 {
		resp.Status.StatusCode = int(follow.Error.Code)
		resp.Status.StatusMsg = follow.Error.Message
		return resp, nil
	}

	resp.Status = code.SUCCEED

	return resp, nil
}
