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
	"github.com/zeromicro/go-zero/core/stores/redis"
)

const (
	prefixPsw = "password#"
	expirePsw = 60 * 5 // 密码过期时间
)

type RegisterLogicLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogicLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogicLogic {
	return &RegisterLogicLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogicLogic) RegisterLogic(req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	resp = new(types.RegisterResponse)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.VerificationCode = strings.TrimSpace(req.VerificationCode)

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

	// 检查验证码
	rdbCode, err := l.svcCtx.BizRedis.Get(prefixVC + req.Email)
	if err == redis.Nil {
		resp.Status = code.VerificationCodeIsEmpty
		return resp, nil
	} else if err != nil {
		logx.Errorf("get verification code cd failed: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	} else if rdbCode != req.VerificationCode {
		resp.Status = code.WrongVerificationCode
		return resp, nil
	}
	if _, err = l.svcCtx.BizRedis.Del(prefixVC + req.Email); err != nil {
		logx.Errorf("del verification code failed: %v", err)
	}

	// 密码加密
	hashedPwd, err := helper.EncryptPassword(req.Password)
	if err != nil {
		logx.Errorf("encrypt password failed: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}

	// 调用用户服务注册
	user, err := l.svcCtx.UserRpc.Register(l.ctx, &userclient.RegisterRequest{
		Email:    req.Email,
		Password: hashedPwd,
	})
	if err != nil {
		logx.Errorf("register failed: %v", err)
		resp.Status.StatusCode = int(user.Error.Code)
		resp.Status.StatusMsg = user.Error.Message
		return resp, nil
	}

	// 生成token
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

	resp = &types.RegisterResponse{
		Status: code.SUCCEED,
		Token: types.Token{
			AccessToken: token.AccessToken,
			ExpireAt:    token.ExpireAt,
		},
		UserID: int(user.Id),
	}

	// 用户密码存入redis
	if err = l.svcCtx.BizRedis.Setex(prefixPsw+req.Email, hashedPwd, expirePsw); err != nil {
		logx.Errorf("set password cache failed: %v", err)
	}

	return resp, nil
}
