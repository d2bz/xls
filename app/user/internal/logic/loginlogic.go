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

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *user.LoginRequest) (resp *user.LoginResponse, err error) {
	resp = new(user.LoginResponse)

	// 尝试从redis读取user信息
	var u *model.User
	userStr, err := l.svcCtx.BizRedis.Get(prefixUser + in.Email)
	if err != nil {
		logx.Errorf("get user cache failed: %v", err)
	}
	if err = u.FromString(userStr); err != nil {
		logx.Errorf("failed to convert string to user: %v", err)
	} else {
		// redis不存在则查询数据库
		db := l.svcCtx.MysqlDB
		u, err = model.GetUserByEmail(db, in.Email)
		if u == nil {
			resp.Error = code.UserNotFound
			return
		} else if err != nil {
			logx.Errorf("Get user by email failed: %v", err)
			return
		}
	}

	// 生成token
	token, err := helper.BuildToken(&helper.TokenOptions{
		AccessSecret: l.svcCtx.Config.Auth.AccessSecret,
		AccessExpire: l.svcCtx.Config.Auth.AccessExpire,
		UserID:       int(u.ID),
	})
	if err != nil {
		logx.Errorf("build token failed: %v", err)
		resp.Error = code.FAILED
		return
	}

	return &user.LoginResponse{
		Error: code.SUCCEED,
		Token: &user.Token{
			AccessToken: token.AccessToken,
			ExpireAt:    token.ExpireAt,
		},
		Id: int64(u.ID),
	}, nil
}
