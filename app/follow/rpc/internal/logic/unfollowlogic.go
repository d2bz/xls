package logic

import (
	"context"
	"gorm.io/gorm"
	"strconv"
	"xls/app/follow/rpc/internal/code"
	"xls/app/follow/rpc/internal/model"
	"xls/app/follow/rpc/internal/types"

	"xls/app/follow/rpc/follow"
	"xls/app/follow/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnFollowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUnFollowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnFollowLogic {
	return &UnFollowLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UnFollowLogic) UnFollow(in *follow.UnFollowRequest) (*follow.UnFollowResponse, error) {
	resp := new(follow.UnFollowResponse)
	f, err := model.FollowFindByUserIDAndFollowedUserID(l.svcCtx.MysqlDB, in.UserID, in.FollowedUserID)
	if err != nil {
		l.Logger.Errorf("[Unfollow] find follow by rpc and followed rpc error: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}
	if f.FollowStatus != types.FollowStatusFollow {
		resp.Error = code.FollowStatusError
		return resp, nil
	}

	db := l.svcCtx.MysqlDB
	err = db.Transaction(func(tx *gorm.DB) error {
		err := model.FollowUpdateFields(tx, f.ID, map[string]interface{}{
			"follow_status": types.FollowStatusUnfollow,
		})
		if err != nil {
			return err
		}

		err = model.DecrFollowCount(tx, in.UserID)
		if err != nil {
			return err
		}

		return model.DecrFansCount(tx, in.FollowedUserID)
	})
	if err != nil {
		l.Logger.Errorf("[Unfollow] transaction err: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	_, err = l.svcCtx.BizRedis.ZremCtx(l.ctx, userFollowKey(in.UserID), strconv.FormatUint(in.FollowedUserID, 10))
	if err != nil {
		l.Logger.Errorf("[UnFollow] redis zrem follows err: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	_, err = l.svcCtx.BizRedis.ZremCtx(l.ctx, userFansKey(in.FollowedUserID), strconv.FormatUint(in.UserID, 10))
	if err != nil {
		l.Logger.Errorf("[UnFollow] redis zrem fans err: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	resp.Error = code.SUCCEED

	return resp, nil
}
