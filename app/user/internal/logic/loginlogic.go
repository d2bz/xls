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
	u := new(model.User)
	userStr, err := l.svcCtx.BizRedis.Get(prefixUser + in.Email)
	if err != nil {
		l.Logger.Errorf("get user cache failed: %v", err)
	}
	if err = u.FromString(userStr); err != nil {
		l.Logger.Errorf("failed to convert string to user: %v", err)
	} else {
		// redis不存在则查询数据库
		db := l.svcCtx.MysqlDB
		u, err = model.GetUserByEmail(db, in.Email)
		if u == nil {
			resp.Error = code.UserNotFound
			return resp, nil
		} else if err != nil {
			l.Logger.Errorf("Get user by email failed: %v", err)
			resp.Error = code.FAILED
			return resp, nil
		}
	}

	// 生成token
	token, err := helper.BuildToken(&helper.TokenOptions{
		AccessSecret: l.svcCtx.Config.Auth.AccessSecret,
		AccessExpire: l.svcCtx.Config.Auth.AccessExpire,
		UserID:       int(u.ID),
	})
	if err != nil {
		l.Logger.Errorf("build token failed: %v", err)
		resp.Error = code.FAILED
		return resp, nil
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
