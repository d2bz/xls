package logic

import (
	"context"
	"strings"

	"xls/app/core/internal/code"
	"xls/app/core/internal/helper"
	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"
	"xls/app/user/userclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) LoginLogic(req *types.LoginRequest) (resp *types.LoginResponse, err error) {
	resp = new(types.LoginResponse)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)

	// 验证输入格式
	matched := helper.CheckEmailFormat(req.Email)
	if !matched {
		resp.Status = code.EmailFormatErorr
		return resp, nil
	}
	matched = helper.CheckPasswordFormat(req.Password)
	if !matched {
		resp.Status = code.PasswordFormatError
		return resp, nil
	}

	// 密码加密
	hashedPwd, err := helper.EncryptPassword(req.Password)
	if err != nil {
		l.Logger.Errorf("encrypt password failed: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}

	// 调用用户登录服务
	user, err := l.svcCtx.UserRpc.Login(l.ctx, &userclient.LoginRequest{
		Email:    req.Email,
		Password: hashedPwd,
	})
	if err != nil {
		l.Logger.Errorf("login rpc failed: %v", err)
		return resp, nil
	}
	if user.Error.Code != 0 {
		resp.Status.StatusCode = int(user.Error.Code)
		resp.Status.StatusMsg = user.Error.Message
		return resp, nil
	}

	resp = &types.LoginResponse{
		Status: code.SUCCEED,
		UserID: int(user.Id),
		Token: types.Token{
			AccessToken: user.Token.AccessToken,
			ExpireAt:    user.Token.ExpireAt,
		},
	}
	return resp, nil
}
