package cron

import (
	"context"
	"fmt"
	"github.com/go-co-op/gocron/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"strings"
	"time"
	"xls/app/like/rpc/internal/svc"
	"xls/app/like/rpc/internal/types"
)

func ScheduledTask(ctx *svc.ServiceContext) {
	s, err := gocron.NewScheduler()
	if err != nil {
		logx.Error(err)
		panic(err)
	}
	cbg := context.Background()
	j, err := s.NewJob(
		gocron.DurationJob(
			5*time.Minute,
		),
		gocron.NewTask(func() {
			genHotVideoCache(cbg, ctx)
		}),
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(j.ID())

	s.Start()
}

func genHotVideoCache(cbg context.Context, ctx *svc.ServiceContext) {
	now := time.Now().UnixMilli()
	oneDayAgo := now - types.MillisecondsPerDay

	var (
		cursor uint64
		keys   []string
	)

	for {
		res, nextCursor, err := ctx.BizRedis.ScanCtx(cbg, cursor, "like#video#*", 100)
		if err != nil {
			logx.Errorf("[cron] redis scan error: %v", err)
			return
		}
		keys = append(keys, res...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	likeKey, hotKey, tempKey := types.LikeKey, types.HotKey, types.TempHotKey
	var zs []redis.Z

	for _, key := range keys {
		videoID := strings.TrimPrefix(key, likeKey)
		count, err := ctx.BizRedis.ZcountCtx(cbg, key, oneDayAgo, now)
		if err != nil {
			logx.Errorf("[cron] redis zcount key %s error: %v", key, err)
			continue
		}
		if count > 0 {
			zs = append(zs, redis.Z{
				Score:  float64(count),
				Member: videoID,
			})
		}
	}
	if len(zs) == 0 {
		logx.Info("[cron] no hot video cache")
		return
	}

	pipe, err := ctx.BizRedis.TxPipeline()
	if err != nil {
		logx.Errorf("[cron] redis tx_pipeline error: %v", err)
		return
	}
	pipe.Del(cbg, tempKey)
	pipe.ZAdd(cbg, tempKey, zs...)
	pipe.Expire(cbg, tempKey, 10*time.Minute)
	_, err = pipe.Exec(cbg)
	if err != nil {
		logx.Errorf("[cron] redis exec error: %v", err)
		return
	}

	_, err = ctx.BizRedis.EvalCtx(cbg, `redis.call('RENAME', KEYS[1], KEYS[2])`, []string{tempKey, hotKey})
	if err != nil {
		logx.Errorf("[cron] redis eval error: %v", err)
		return
	}
}
