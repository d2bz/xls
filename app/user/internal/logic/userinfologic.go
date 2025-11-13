package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zeromicro/go-zero/core/threading"
	"xls/app/user/internal/code"
	"xls/app/user/internal/model"
	"xls/app/user/internal/types"

	"xls/app/user/internal/svc"
	"xls/app/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

const ()

type UserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserInfoLogic {
	return &UserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UserInfoLogic) UserInfo(in *user.UserInfoRequest) (*user.UserInfoResponse, error) {
	resp := new(user.UserInfoResponse)

	key := UserInfoKey(in.UserID)
	infoString, err := l.svcCtx.BizRedis.GetCtx(l.ctx, key)
	if err == nil && infoString != "" {
		u := new(model.User)
		if err := json.Unmarshal([]byte(infoString), u); err == nil {
			_ = l.svcCtx.BizRedis.ExpireCtx(l.ctx, key, types.UserInfoExpire)
			l.Logger.Infof("[userInfo] user info cache hit: %+v", u)
			resp = &user.UserInfoResponse{
				Error:  code.SUCCEED,
				UserId: uint64(u.ID),
				Email:  u.Email,
				Name:   u.Name,
				Avatar: u.Avatar,
			}
			return resp, nil
		}
	}

	userModel := model.NewUserModel(l.svcCtx.MysqlDB)
	userData, err := userModel.FindUserByID(in.UserID)
	if err != nil {
		l.Logger.Errorf("[userInfo] userModel.FindUserByID error: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	threading.GoSafe(func() {
		userJson, _ := json.Marshal(userData)
		if err := l.svcCtx.BizRedis.SetexCtx(l.ctx, key, string(userJson), types.UserInfoExpire); err != nil {
			l.Logger.Errorf("[userInfo] set user info cache error: %v", err)
		}
	})

	resp = &user.UserInfoResponse{
		Error:  code.SUCCEED,
		UserId: uint64(userData.ID),
		Email:  userData.Email,
		Name:   userData.Name,
		Avatar: userData.Avatar,
	}

	return resp, nil
}

func UserInfoKey(userID uint64) string {
	return fmt.Sprintf("user:info:%d", userID)
}
