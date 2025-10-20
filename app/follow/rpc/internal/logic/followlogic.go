package logic

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"strconv"
	"time"
	"xls/app/follow/rpc/internal/code"
	"xls/app/follow/rpc/internal/model"
	"xls/app/follow/rpc/internal/types"

	"xls/app/follow/rpc/follow"
	"xls/app/follow/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type FollowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFollowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowLogic {
	return &FollowLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FollowLogic) Follow(in *follow.FollowRequest) (*follow.FollowResponse, error) {
	resp := new(follow.FollowResponse)
	f, err := model.FollowFindByUserIDAndFollowedUserID(l.svcCtx.MysqlDB, in.UserID, in.FollowedUserID)
	if err != nil {
		l.Logger.Errorf("[Follow] find follow by user and followed user error: %v req: %v", err, in)
		resp.Error = code.FAILED
		return resp, nil
	}

	if f.FollowStatus != types.FollowStatusUnfollow {
		resp.Error = code.FollowStatusError
		return resp, nil
	}

	db := l.svcCtx.MysqlDB

	err = db.Transaction(func(tx *gorm.DB) error {
		if f != nil {
			err = model.FollowUpdateFields(tx, f.ID, map[string]interface{}{
				"follow_status": types.FollowStatusFollow,
			})
		} else {
			err = model.FollowInsert(tx, &model.Follow{
				UserID:         in.UserID,
				FollowedUserID: in.FollowedUserID,
				FollowStatus:   types.FollowStatusFollow,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			})
		}
		if err != nil {
			return err
		}

		err = model.IncrFollowCount(tx, in.UserID)
		if err != nil {
			return err
		}
		return model.IncrFansCount(tx, in.FollowedUserID)
	})

	if err != nil {
		l.Logger.Errorf("[Follow] transaction err: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	_, err = l.svcCtx.BizRedis.ZaddCtx(l.ctx, userFollowKey(in.UserID), time.Now().UnixMilli(), strconv.FormatUint(in.FollowedUserID, 10))
	if err != nil {
		l.Logger.Errorf("[Follow] redis zadd follow err: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	_, err = l.svcCtx.BizRedis.ZremrangebyrankCtx(l.ctx, userFollowKey(in.UserID), 0, -(types.CacheMaxFollowCount))
	if err != nil {
		l.Logger.Errorf("[Follow] redis zremrangebyrank err: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	_, err = l.svcCtx.BizRedis.ZaddCtx(l.ctx, userFansKey(in.FollowedUserID), time.Now().UnixMilli(), strconv.FormatUint(in.UserID, 10))
	if err != nil {
		l.Logger.Errorf("[Follow] redis zadd fans err: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	_, err = l.svcCtx.BizRedis.ZremrangebyrankCtx(l.ctx, userFansKey(in.FollowedUserID), 0, -(types.CacheMaxFansCount))
	if err != nil {
		l.Logger.Errorf("[Follow] redis zremrangebyrank err: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	resp.Error = code.SUCCEED

	return resp, nil
}

func userFollowKey(userID uint64) string {
	return fmt.Sprintf("user#follow#%v", userID)
}

func userFansKey(userID uint64) string {
	return fmt.Sprintf("user#fans#%v", userID)
}
