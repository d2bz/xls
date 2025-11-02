package logic

import (
	"context"
	"github.com/zeromicro/go-zero/core/threading"
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
	followCountModel := model.NewFollowCountModel(l.svcCtx.MysqlDB)

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
		lastID          uint64
		cursor          int64
		followedUserIDs []uint64
		follows         []*model.Follow
		curPage         []*follow.FollowItem
	)

	followedUserIDsFromCache, _ := l.cacheFollowedUserIDs(l.ctx, in.UserID, in.Cursor, in.PageSize)
	if len(followedUserIDsFromCache) > 0 {
		isCache = true
		if followedUserIDsFromCache[len(followedUserIDsFromCache)-1] == -1 {
			followedUserIDsFromCache = followedUserIDsFromCache[:len(followedUserIDsFromCache)-1]
			isEnd = true
		}
		if len(followedUserIDsFromCache) == 0 {
			resp.Error = code.SUCCEED
			return resp, nil
		}
		follows, err = followModel.FindByFollowedUserIDs(l.ctx, in.UserID, followedUserIDsFromCache)
		if err != nil {
			l.Logger.Errorf("[followList] followModel.FindByFollowedUserIDs err: %v", err)
			resp.Error = code.FAILED
			return resp, nil
		}

		// 这一次遍历是用id过滤掉cursor值相同的记录，防止返回重复记录
		for k, followItem := range follows {
			if followItem.CreatedAt.Unix() == in.Cursor && followItem.ID == in.ID {
				follows = follows[k:]
				break
			}
		}

		for _, followItem := range follows {
			// 这里是对数据一致性的保证，确保用户id在数据库里都是存在的
			followedUserIDs = append(followedUserIDs, followItem.FollowedUserID)
			curPage = append(curPage, &follow.FollowItem{
				ID:             followItem.ID,
				FollowedUserID: followItem.FollowedUserID,
				CreateTime:     followItem.CreatedAt.Unix(),
			})
		}
	} else {
		follows, err = followModel.FindByUserID(l.ctx, in.UserID, types.CacheMaxFollowCount)
		if err != nil {
			l.Logger.Errorf("[followList] followModel.FindByUserID err: %v", err)
			resp.Error = code.FAILED
			return resp, nil
		}
		if len(follows) == 0 {
			resp.Error = code.SUCCEED
			return resp, nil
		}
		// 数据库查出全部关注，按游标过滤出当前页，但不影响查出的全部数据，方便后续写回数据库
		// 存在风险：如果用户刚好取关了cursor记录，则会导致返回错误
		var pageFollows []*model.Follow
		for k, followItem := range follows {
			if followItem.CreatedAt.Unix() == in.Cursor && followItem.ID == in.ID {
				// 判断下标是否越界
				if k+int(in.PageSize)+1 < len(follows) {
					pageFollows = follows[k : k+int(in.PageSize)+1]
				} else {
					pageFollows = follows[k:]
					isEnd = true
				}
				break
			}
		}

		// 可能是第一次查询，返回第一页
		if len(pageFollows) == 0 {
			if int(in.PageSize) < len(follows) {
				pageFollows = follows[:in.PageSize]
			} else {
				pageFollows = follows
				isEnd = true
			}
		}

		for _, followItem := range pageFollows {
			followedUserIDs = append(followedUserIDs, followItem.FollowedUserID)
			curPage = append(curPage, &follow.FollowItem{
				ID:             followItem.ID,
				FollowedUserID: followItem.FollowedUserID,
				CreateTime:     followItem.CreatedAt.Unix(),
			})
		}
	}

	if len(curPage) > 0 {
		pageLast := curPage[len(curPage)-1]
		lastID = pageLast.ID
		cursor = pageLast.CreateTime
		if cursor < 0 {
			cursor = 0
		}
	}

	fc, err := followCountModel.FindByUserIDs(l.ctx, followedUserIDs)
	if err != nil {
		l.Logger.Errorf("[followList] followCountModel.FindByUserIDs err: %v", err)
	}
	uidFansCount := make(map[uint64]int)
	for _, f := range fc {
		uidFansCount[f.UserID] = f.FansCount
	}
	for _, cur := range curPage {
		cur.FansCount = int64(uidFansCount[cur.FollowedUserID])
	}

	resp = &follow.FollowListResponse{
		Error:   code.SUCCEED,
		IsEnd:   isEnd,
		Cursor:  cursor,
		ID:      lastID,
		Follows: curPage,
	}

	if !isCache {
		threading.GoSafe(func() {
			if len(follows) < types.CacheMaxFollowCount && len(follows) > 0 {
				follows = append(follows, &model.Follow{FollowedUserID: -1})
			}
			err = l.addCacheFollow(context.Background(), in.UserID, follows)
			if err != nil {
				logx.Errorf("[followList] addCacheFollow err: %v", err)
			}
		})
	}

	return resp, nil
}

// 从缓存中获取关注的用户ID列表：检查键是否存在，存在则更新过期时间。根据游标获取一页的id并返回
func (l *FollowListLogic) cacheFollowedUserIDs(ctx context.Context, userID uint64, cursor, pageSize int64) ([]uint64, error) {
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

func (l *FollowListLogic) addCacheFollow(ctx context.Context, userID uint64, follows []*model.Follow) error {
	if len(follows) == 0 {
		return nil
	}
	key := userFollowKey(userID)
	for _, followItem := range follows {
		var score int64
		if followItem.FollowedUserID == -1 {
			score = 0
		} else {
			score = followItem.CreatedAt.Unix()
		}
		_, err := l.svcCtx.BizRedis.ZaddCtx(ctx, key, score, strconv.FormatUint(followItem.FollowedUserID, 10))
		if err != nil {
			logx.Errorf("[addCacheFollow] BizRedis.Zadd err: %v", err)
			return err
		}
	}

	return l.svcCtx.BizRedis.ExpireCtx(ctx, key, userFollowExpireTime)
}
