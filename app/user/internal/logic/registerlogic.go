package logic

import (
	"context"

	"xls/app/user/internal/code"
	"xls/app/user/internal/helper"
	"xls/app/user/internal/model"
	"xls/app/user/internal/svc"
	"xls/app/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	prefixUser = "user#"
	expireUser = 60 * 5 // 密码过期时间
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
	db := l.svcCtx.MysqlDB
	u, err := model.GetUserByEmail(db, in.Email)
	if u != nil {
		userResp.Error = code.UserAlreadyExists
		return
	} else if err != nil {
		logx.Errorf("Get user by email failed: %v", err)
		return
	}

	// 注册用户
	u = &model.User{
		Email:    in.Email,
		Name:     "user-" + in.Email,
		Password: in.Password,
	}
	if err = u.Insert(db); err != nil {
		logx.Errorf("Insert user failed: %v", err)
		userResp.Error = code.FAILED
		return
	}

	// 生成token
	token, err := helper.BuildToken(&helper.TokenOptions{
		AccessSecret: l.svcCtx.Config.Auth.AccessSecret,
		AccessExpire: l.svcCtx.Config.Auth.AccessExpire,
		UserID:       int(u.ID),
	})
	if err != nil {
		logx.Errorf("build token failed: %v", err)
		userResp.Error = code.FAILED
		return
	}

	// user信息存入redis
	userStr, err := u.ToString()
	if err == nil {
		err = l.svcCtx.BizRedis.Setex(prefixUser+in.Email, userStr, expireUser)
		if err != nil {
			logx.Errorf("set user cache failed: %v", err)
		}
	} else {
		logx.Errorf("failed to convert user to string: %v", err)
	}

	userResp = &user.RegisterResponse{
		Error: code.SUCCEED,
		Token: &user.Token{
			AccessToken: token.AccessToken,
			ExpireAt:    token.ExpireAt,
		},
		Id: int64(u.ID),
	}
	return
}
