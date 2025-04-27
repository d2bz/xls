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

type LoginLogicLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogicLogic {
	return &LoginLogicLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogicLogic) LoginLogic(req *types.LoginRequest) (resp *types.LoginResponse, err error) {
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
		logx.Errorf("encrypt password failed: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}

	// 调用用户登录服务
	user, err := l.svcCtx.UserRpc.Login(l.ctx, &userclient.LoginRequest{
		Email:    req.Email,
		Password: hashedPwd,
	})
	if err != nil {
		logx.Errorf("user login failed: %v", err)
		resp.Status.StatusCode = int(user.Error.Code)
		resp.Status.StatusMsg = user.Error.Message
		return resp, nil
	}

	// 生成Token
	token, err := helper.BuildToken(&helper.TokenOptions{
		AccessSecret: l.svcCtx.Config.Auth.AccessSecret,
		AccessExpire: l.svcCtx.Config.Auth.AccessExpire,
		UserID:       int(user.Id),
	})
	if err != nil {
		logx.Errorf("build token failed: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}

	resp = &types.LoginResponse{
		Status: code.SUCCEED,
		UserID: int(user.Id),
		Token: types.Token{
			AccessToken: token.AccessToken,
			ExpireAt:    token.ExpireAt,
		},
	}
	return resp, nil
}
