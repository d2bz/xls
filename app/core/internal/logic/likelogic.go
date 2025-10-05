package logic

import (
	"context"
	"encoding/json"

	"xls/app/core/internal/code"
	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"
	"xls/app/like/rpc/like"

	"github.com/zeromicro/go-zero/core/logx"
)

type LikeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LikeLogic) Like(req *types.LikeRequest) (resp *types.LikeResponse, err error) {
	resp = new(types.LikeResponse)

	uid, err := l.ctx.Value("userid").(json.Number).Int64()
	if err != nil {
		resp.Status = code.NoLogin
		return resp, nil
	}

	like, err := l.svcCtx.LikeRpc.Like(l.ctx, &like.LikeRequest{
		UserID:     uint64(uid),
		TargetID:   req.TargetID,
		TargetType: req.TargetType,
	})
	if err != nil {
		resp.Status = code.FAILED
		l.Logger.Errorf("like rpc failed: %v", err)
		return resp, nil
	}
	if like.Error.Code != 0 {
		resp.Status.StatusCode = int(like.Error.Code)
		resp.Status.StatusMsg = like.Error.Message
		return resp, nil
	}

	resp.Status = code.SUCCEED

	return resp, nil
}
