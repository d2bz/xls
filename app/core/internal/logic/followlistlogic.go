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

type FollowListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewFollowListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowListLogic {
	return &FollowListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FollowListLogic) FollowList(req *types.FollowListRequest) (resp *types.FollowListResponse, err error) {
	resp = new(types.FollowListResponse)

	userID, err := l.ctx.Value("user_id").(json.Number).Int64()
	if err != nil {
		resp.Status = code.NoLogin
		return resp, nil
	}

	list, err := l.svcCtx.FollowRpc.FollowList(l.ctx, &followclient.FollowListRequest{
		ID:       req.ID,
		UserID:   uint64(userID),
		Cursor:   req.Cursor,
		PageSize: req.PageSize,
	})
	if err != nil {
		l.Logger.Errorf("followlist err: %v", err)
		resp.Status = code.FAILED
		return resp, nil
	}
	if list.Error.Code != 0 {
		resp.Status.StatusCode = int(list.Error.Code)
		resp.Status.StatusMsg = list.Error.Message
		return resp, nil
	}

	followList := make([]types.FollowItem, 0, len(list.Follows))
	for _, followItem := range list.Follows {
		followList = append(followList, types.FollowItem{
			ID:             followItem.ID,
			FollowedUserID: followItem.FollowedUserID,
			FansCount:      followItem.FansCount,
			CreateTime:     followItem.CreateTime,
		})
	}

	resp = &types.FollowListResponse{
		Status:  code.SUCCEED,
		Follows: followList,
		Cursor:  list.Cursor,
		IsEnd:   list.IsEnd,
		ID:      list.ID,
	}

	return resp, nil
}
