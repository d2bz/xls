package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"strconv"
	"strings"
	"xls/app/user/rpc/userclient"
	"xls/app/video/mq/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"xls/app/video/mq/internal/svc"
)

type VideoMqLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewVideoMqLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VideoMqLogic {
	return &VideoMqLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *VideoMqLogic) Consume(_ context.Context, _, val string) error {
	msg := &types.CanalVideoMsg{}
	err := json.Unmarshal([]byte(val), msg)
	if err != nil {
		l.Logger.Errorf("[video-mq] unmarshal msg: %+v err: %v", val, err)
		return err
	}

	return l.VideoOperate(msg)
}

func (l *VideoMqLogic) VideoOperate(msg *types.CanalVideoMsg) error {
	if len(msg.Data) == 0 {
		return nil
	}

	var esData []*types.EsVideoMsg
	for _, v := range msg.Data {
		videoID, _ := strconv.ParseUint(v.ID, 10, 64)
		authorID, _ := strconv.ParseUint(v.Uid, 10, 64)
		likeNum, _ := strconv.ParseInt(v.LikeNum, 10, 64)
		commentNum, _ := strconv.ParseInt(v.CommentNum, 10, 64)

		userInfo, err := l.svcCtx.UserRPC.UserInfo(l.ctx, &userclient.UserInfoRequest{
			UserID: authorID,
		})
		if err != nil {
			l.Logger.Errorf("[video-mq] get user info err: %v", err)
			return err
		}
		if userInfo.Error.Code != 0 {
			l.Logger.Errorf("[video-mq] get user info err: %v", userInfo.Error.Message)
			return errors.New(fmt.Sprint(userInfo.Error.Message))
		}

		esData = append(esData, &types.EsVideoMsg{
			VideoID:      videoID,
			Title:        v.Title,
			Url:          v.Url,
			AuthorID:     authorID,
			AuthorName:   userInfo.Name,
			AuthorAvatar: userInfo.Avatar,
			LikeNum:      likeNum,
			CommentNum:   commentNum,
			CreatedAt:    v.CreatedAt,
			UpdatedAt:    v.UpdatedAt,
			DeletedAt:    v.DeletedAt,
		})
	}
	err := l.ButchUpsertToES(l.ctx, esData)
	if err != nil {
		l.Logger.Errorf("[video-mq] upsert to es error: %v", err)
	}

	return err
}

func (l *VideoMqLogic) ButchUpsertToES(ctx context.Context, data []*types.EsVideoMsg) error {
	if len(data) == 0 {
		return nil
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client: l.svcCtx.Es,
		Index:  "video-index",
	})
	if err != nil {
		return err
	}
	for _, v := range data {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}

		payload := fmt.Sprintf(`{"doc":%s,"doc_as_upsert":true}`, string(b))
		err = bi.Add(ctx, esutil.BulkIndexerItem{
			Action:     "update",
			DocumentID: fmt.Sprintf("%d", v.VideoID),
			Body:       strings.NewReader(payload),
			OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, item2 esutil.BulkIndexerResponseItem) {
			},
			OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, item2 esutil.BulkIndexerResponseItem, err error) {
			},
		})
		if err != nil {
			return err
		}
	}

	return bi.Close(ctx)
}
