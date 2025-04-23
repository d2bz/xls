package logic

import (
	"context"

	"xls/app/user/internal/code"
	"xls/app/user/internal/svc"
	"xls/app/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterLogic) Register(in *user.RegisterRequest) (userResp *user.RegisterResponse, err error) {
	userResp = new(user.RegisterResponse)
	// 检查用户是否存在
	userResp.Error = code.UserIsExist
	return
}
