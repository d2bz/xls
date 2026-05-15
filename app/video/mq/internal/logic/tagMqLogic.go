package logic

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-zero/core/logx"
	"xls/app/video/mq/internal/svc"
	"xls/app/video/mq/internal/types"
)

type TagMqLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewTagMqLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TagMqLogic {
	return &TagMqLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *TagMqLogic) Consume(_ context.Context, _, val string) error {
	msg := &types.CanalTagMsg{}
	if err := json.Unmarshal([]byte(val), msg); err != nil {
		l.Logger.Errorf("[tag-mq] unmarshal msg: %+v err: %v", val, err)
		return err
	}

	if msg.Type != "INSERT" || len(msg.Data) == 0 {
		return nil
	}

	var tagNames []string
	for _, d := range msg.Data {
		if d.TagName == "" {
			continue
		}
		tagNames = append(tagNames, d.TagName)
	}
	if len(tagNames) == 0 {
		return nil
	}

	vectors, err := l.svcCtx.Embedder.EmbedStrings(l.ctx, tagNames)
	if err != nil {
		l.Logger.Errorf("[tag-mq] embed tag_names failed: %v", err)
		return err
	}

	floatVectors := make([][]float32, len(vectors))
	for i, v := range vectors {
		f := make([]float32, len(v))
		for j, fv := range v {
			f[j] = float32(fv)
		}
		floatVectors[i] = f
	}

	if err := l.svcCtx.Milvus.InsertTag(l.ctx, tagNames, floatVectors); err != nil {
		l.Logger.Errorf("[tag-mq] insert to milvus failed: %v", err)
		return err
	}

	l.Logger.Infof("[tag-mq] inserted %d tags to milvus", len(tagNames))
	return nil
}
