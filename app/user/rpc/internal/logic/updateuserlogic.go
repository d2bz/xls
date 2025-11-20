package logic

import (
	"context"
	"xls/app/user/rpc/internal/code"
	"xls/app/user/rpc/internal/model"

	"xls/app/user/rpc/internal/svc"
	"xls/app/user/rpc/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserLogic {
	return &UpdateUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateUserLogic) UpdateUser(in *user.UpdateUserRequest) (*user.UpdateUserResponse, error) {
	resp := &user.UpdateUserResponse{}
	userModel := model.NewUserModel(l.svcCtx.MysqlDB)

	updateData := make(map[string]any)

	if in.Name != "" {
		updateData["name"] = in.Name
	}
	if in.Avatar != "" {
		updateData["avatar"] = in.Avatar
	}

	err := userModel.UpdateFields(l.ctx, in.Id, updateData)
	if err != nil {
		l.Logger.Errorf("[updateUser] UpdateUser err: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	resp.Error = code.SUCCEED

	return resp, nil
}
