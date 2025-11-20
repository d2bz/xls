package logic

import (
	"context"
	"encoding/json"
	"xls/app/core/internal/code"
	"xls/app/user/rpc/userclient"

	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserLogic {
	return &UpdateUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserLogic) UpdateUser(req *types.UpdateUserRequest) (resp *types.UpdateUserResponse, err error) {
	resp = &types.UpdateUserResponse{}

	userID, err := l.ctx.Value("user_id").(json.Number).Int64()
	if err != nil {
		resp.Status = code.NoLogin
		return resp, nil
	}

	res, err := l.svcCtx.UserRpc.UpdateUser(l.ctx, &userclient.UpdateUserRequest{
		Id:     uint64(userID),
		Name:   req.Name,
		Avatar: req.Avatar,
	})
	if err != nil {
		l.Logger.Errorf("update user err: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}
	if res.Error.Code != 0 {
		resp.Status.StatusCode = int(res.Error.Code)
		resp.Status.StatusMsg = res.Error.Message
		return resp, nil
	}

	resp.Status = code.SUCCEED

	return resp, nil
}
