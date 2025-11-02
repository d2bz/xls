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

const userFansExpireTime = 3600 * 24 * 2

type FansListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFansListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FansListLogic {
	return &FansListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FansListLogic) FansList(in *follow.FansListRequest) (*follow.FansListResponse, error) {
	resp := new(follow.FansListResponse)
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
		err            error
		isCache, isEnd bool
		lastFansID     uint64
		cursor         int64
		fansUserIDs    []uint64
		fansModel      []*model.Follow
		curPage        []*follow.FansItem
	)

	fansIDs, createTimes, _ := l.cacheFansUserIDs(l.ctx, in.UserID, in.Cursor, in.PageSize)
	if len(fansIDs) > 0 {
		isCache = true
		if fansIDs[len(fansIDs)-1] == -1 {
			fansIDs = fansIDs[:len(fansIDs)-1]
			isEnd = true
		}
		if len(fansIDs) == 0 {
			resp.Error = code.SUCCEED
			return resp, nil
		}

		fansUserIDs = fansIDs
		for i, id := range fansIDs {
			curPage = append(curPage, &follow.FansItem{
				FansUserID: id,
				CreateTime: createTimes[i],
			})
		}

		for k, fansItem := range curPage {
			if fansItem.CreateTime == in.Cursor && fansItem.FansUserID == in.LastFansID {
				curPage = curPage[k:]
				break
			}
		}
	} else {
		fansModel, err = followModel.FindByFollowedUserID(l.ctx, in.UserID, types.CacheMaxFansCount)
		if err != nil {
			l.Logger.Errorf("[fansList] FindByFollowedUserID err: %v", err)
			resp.Error = code.FAILED
			return resp, nil
		}
		if len(fansModel) == 0 {
			resp.Error = code.SUCCEED
			return resp, nil
		}

		var pageFans []*model.Follow
		for k, fansItem := range fansModel {
			if fansItem.CreatedAt.Unix() == in.Cursor && fansItem.UserID == in.LastFansID {
				if k+int(in.PageSize)+1 < len(fansModel) {
					pageFans = fansModel[k : k+int(in.PageSize)+1]
				} else {
					pageFans = fansModel[k:]
					isEnd = true
				}
				break
			}
		}

		if len(pageFans) == 0 {
			if int(in.PageSize) < len(fansModel) {
				pageFans = fansModel[:in.PageSize]
			} else {
				pageFans = fansModel
				isEnd = true
			}
		}

		for _, fansItem := range pageFans {
			fansUserIDs = append(fansUserIDs, fansItem.FollowedUserID)
			curPage = append(curPage, &follow.FansItem{
				FansUserID: fansItem.FollowedUserID,
				CreateTime: fansItem.CreatedAt.Unix(),
			})
		}

		if len(curPage) > 0 {
			pageLast := curPage[len(curPage)-1]
			lastFansID = pageLast.FansUserID
			cursor = pageLast.CreateTime
			if cursor < 0 {
				cursor = 0
			}
		}

		fc, err := followCountModel.FindByUserIDs(l.ctx, fansUserIDs)
		if err != nil {
			l.Logger.Errorf("[fansList] FindByUserIDs err: %v", err)
		}
		uidFansCount := make(map[uint64]int)
		uidFollowCount := make(map[uint64]int)
		for _, f := range fc {
			uidFansCount[f.UserID] = f.FansCount
			uidFollowCount[f.UserID] = f.FollowCount
		}
		for _, cur := range curPage {
			cur.FansCount = int64(uidFansCount[cur.FansUserID])
			cur.FollowCount = int64(uidFollowCount[cur.FansUserID])
		}

		resp = &follow.FansListResponse{
			Error:      code.SUCCEED,
			Cursor:     cursor,
			Fans:       curPage,
			IsEnd:      isEnd,
			LastFansID: lastFansID,
		}
	}

	if !isCache {
		threading.GoSafe(func() {
			if len(fansModel) < types.CacheMaxFansCount && len(fansModel) > 0 {
				fansModel = append(fansModel, &model.Follow{UserID: -1})
			}
			err = l.addCacheFans(context.Background(), in.UserID, fansModel)
			if err != nil {
				logx.Error("[fansList] addCacheFans err: %v", err)
			}
		})
	}

	return resp, nil
}

func (l *FansListLogic) cacheFansUserIDs(ctx context.Context, userID uint64, cursor, pageSize int64) ([]uint64, []int64, error) {
	key := userFansKey(userID)
	b, err := l.svcCtx.BizRedis.ExistsCtx(ctx, key)
	if err != nil {
		logx.Errorf("[cacheFansUserIDs] redis exists err: %v", err)
	}
	if b {
		err = l.svcCtx.BizRedis.ExpireCtx(ctx, key, userFansExpireTime)
		if err != nil {
			logx.Errorf("[cacheFansUserIDs] redis expire err: %v", err)
		}
	}
	pairs, err := l.svcCtx.BizRedis.ZrevrangebyscoreWithScoresAndLimitCtx(ctx, key, 0, cursor, 0, int(pageSize))
	if err != nil {
		logx.Errorf("[cacheFansUserIDs] redis zrevrangebyscore err: %v", err)
		return nil, nil, err
	}
	var uids []uint64
	var createTimes []int64
	for _, pair := range pairs {
		uid, err := strconv.ParseUint(pair.Key, 10, 64)
		if err != nil {
			logx.Errorf("[[acheFansUserIDs] parse err: %v", err)
			continue
		}
		uids = append(uids, uid)
		createTimes = append(createTimes, pair.Score)
	}
	return uids, createTimes, nil
}

func (l *FansListLogic) addCacheFans(ctx context.Context, userID uint64, fans []*model.Follow) error {
	if len(fans) == 0 {
		return nil
	}
	key := userFansKey(userID)
	for _, fan := range fans {
		var score int64
		if fan.UserID == -1 {
			score = 0
		} else {
			score = fan.CreatedAt.Unix()
		}
		_, err := l.svcCtx.BizRedis.ZaddCtx(ctx, key, score, strconv.FormatUint(fan.UserID, 10))
		if err != nil {
			logx.Errorf("[addCacheFans] redis add err: %v", err)
			return err
		}
	}

	return l.svcCtx.BizRedis.ExpireCtx(ctx, key, userFansExpireTime)
}
