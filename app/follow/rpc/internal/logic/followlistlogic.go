package logic

import (
	"context"
	"strconv"
	"time"
	"xls/app/follow/rpc/internal/code"
	"xls/app/follow/rpc/internal/model"
	"xls/app/follow/rpc/internal/types"

	"xls/app/follow/rpc/follow"
	"xls/app/follow/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const userFollowExpireTime = 3600 * 24 * 2

type FollowListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFollowListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowListLogic {
	return &FollowListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FollowListLogic) FollowList(in *follow.FollowListRequest) (*follow.FollowListResponse, error) {
	resp := new(follow.FollowListResponse)
	followModel := model.NewFollowModel(l.svcCtx.MysqlDB)

	if in.UserID == 0 {
		resp.Error = code.UserIDIsEmpty
		return resp, nil
	}

	if in.PageSize == 0 {
		in.PageSize = types.DefaultPageSize
	}
	if in.Cursor == 0 {
		in.Cursor = time.Now().Unix()
	}

	var (
		err             error
		isCache, isEnd  bool
		lastID, cursor  int64
		followedUserIDs []uint64
		follows         []*model.Follow
		curPage         []*follow.FollowItem
	)

	return resp, nil
}

// 从缓存中获取关注的用户ID列表：检查键是否存在，存在则更新过期时间。根据游标获取一页的id并返回
func (l *FollowListLogic) cacheFollowUserIDs(ctx context.Context, userID uint64, cursor, pageSize int64) ([]uint64, error) {
	key := userFollowKey(userID)
	b, err := l.svcCtx.BizRedis.ExistsCtx(ctx, key)
	if err != nil {
		logx.Errorf("[cacheFollowUserIDs] redis exists err: %v", err)
	}
	if b {
		err = l.svcCtx.BizRedis.ExpireCtx(ctx, key, userFollowExpireTime)
		if err != nil {
			logx.Errorf("[cacheFollowUserIDs] redis expire err: %v", err)
		}
	}
	pairs, err := l.svcCtx.BizRedis.ZrevrangebyscoreWithScoresAndLimitCtx(ctx, key, 0, cursor, 0, int(pageSize))
	if err != nil {
		logx.Errorf("[cacheFollowUserIds] BizRedis.ZrevrangebyscoreWithScoresAndLimitCtx error: %v", err)
		return nil, err
	}
	var userIDs []uint64
	for _, pair := range pairs {
		userID, err := strconv.ParseUint(pair.Key, 10, 64)
		if err != nil {
			logx.Errorf("[cacheFollowUserIds] strconv.ParseInt error: %v", err)
			continue
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, nil
}
