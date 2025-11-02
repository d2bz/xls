package logic

import (
	"context"
	"encoding/json"
	"xls/app/core/internal/code"
	"xls/app/follow/rpc/followclient"

	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type FansListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFansListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FansListLogic {
	return &FansListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FansListLogic) FansList(req *types.FansListRequest) (resp *types.FansListResponse, err error) {
	resp = new(types.FansListResponse)

	userID, err := l.ctx.Value("user_id").(json.Number).Int64()
	if err != nil {
		resp.Status = code.NoLogin
		return resp, nil
	}

	list, err := l.svcCtx.FollowRpc.FansList(l.ctx, &followclient.FansListRequest{
		UserID:     uint64(userID),
		Cursor:     req.Cursor,
		PageSize:   req.PageSize,
		LastFansID: req.LastFansID,
	})
	if err != nil {
		l.Logger.Errorf("FansList err: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}
	if list.Error.Code != 0 {
		resp.Status.StatusCode = int(list.Error.Code)
		resp.Status.StatusMsg = list.Error.Message
		return resp, nil
	}

	fansList := make([]types.FansItem, 0, len(list.Fans))
	for _, item := range list.Fans {
		fansList = append(fansList, types.FansItem{
			FansUserID:  item.FansUserID,
			FollowCount: item.FollowCount,
			FansCount:   item.FansCount,
			CreateTime:  item.CreateTime,
		})
	}

	resp = &types.FansListResponse{
		Status:     code.SUCCEED,
		Fans:       fansList,
		Cursor:     list.Cursor,
		IsEnd:      list.IsEnd,
		LastFansID: list.LastFansID,
	}

	return resp, nil
}
