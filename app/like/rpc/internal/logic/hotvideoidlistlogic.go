package logic

import (
	"context"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/threading"
	"time"
	"xls/app/like/rpc/internal/code"
	"xls/app/like/rpc/internal/model"
	"xls/app/like/rpc/internal/types"

	"xls/app/like/rpc/internal/svc"
	"xls/app/like/rpc/like"

	"github.com/zeromicro/go-zero/core/logx"
)

type HotVideoIDListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHotVideoIDListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HotVideoIDListLogic {
	return &HotVideoIDListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *HotVideoIDListLogic) HotVideoIDList(in *like.HotVideoIDListRequest) (*like.HotVideoIDListResponse, error) {
	resp := new(like.HotVideoIDListResponse)

	videoIDList, err := l.getHotVideoIDListFromCache()
	if err != nil {
		videoIDList, err = l.getHotVideoIDListFromDB()
		if err != nil || videoIDList == nil {
			l.Logger.Errorf("[hotVideoIDListLogic] getHotVideoIDListFromDB error: %v videoIDList: %v", err, videoIDList)
			resp.Error = code.FAILED
			return resp, nil
		}

		threading.GoSafe(func() {
			l.setTempHotVideoIDListCache(videoIDList)
		})

	}

	resp.VideoIDs = videoIDList
	resp.Error = code.SUCCEED

	return resp, nil
}

func (l *HotVideoIDListLogic) getHotVideoIDListFromCache() ([]string, error) {
	videoIDList, err := l.svcCtx.BizRedis.ZrevrangeCtx(l.ctx, types.HotKey, 0, types.VideoIDsLength-1)
	if err != nil {
		return nil, err
	}

	if len(videoIDList) == 0 {
		return nil, redis.Nil
	}

	return videoIDList, nil
}

func (l *HotVideoIDListLogic) getHotVideoIDListFromDB() ([]string, error) {
	var videoIDList []string
	err := l.svcCtx.MysqlDB.Model(&model.Like{}).
		Select("target_id").
		Where("created_at >= ? AND target_type == ?", time.Now().Add(-24*time.Hour), types.VideoLike).
		Group("target_id").
		Order("COUNT(user_id) DESC").
		Limit(types.VideoIDsLength).
		Pluck("target_id", &videoIDList).Error
	return videoIDList, err
}

func (l *HotVideoIDListLogic) setTempHotVideoIDListCache(videoIDList []string) {
	zs := make([]redis.Z, len(videoIDList))
	for i, videoID := range videoIDList {
		zs[i] = redis.Z{
			Score:  float64(len(videoID) - 1),
			Member: videoID,
		}
	}

	tempKey := types.TempHotKeyDB
	pipe, err := l.svcCtx.BizRedis.TxPipeline()
	if err != nil {
		l.Logger.Errorf("[hotVideoIDSetBack] get redis pipeline err: %v", err)
		return
	}
	pipe.Del(l.ctx, tempKey)
	pipe.ZAdd(l.ctx, tempKey, zs...)
	pipe.Expire(l.ctx, tempKey, 5*time.Minute)
	_, err = pipe.Exec(l.ctx)
	if err != nil {
		l.Logger.Errorf("[hotVideoIDSetBack] redis pipeline exec err: %v", err)
		return
	}

	_, err = l.svcCtx.BizRedis.EvalCtx(l.ctx, `redis.call('RENAME', KEYS[1], KEYS[2])`, []string{tempKey, types.HotKey})
	if err != nil {
		l.Logger.Errorf("[hotVideoIDSetBack] redis eval rename error: %v", err)
		return
	}
}
