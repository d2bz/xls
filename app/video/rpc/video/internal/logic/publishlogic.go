package logic

import (
	"context"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"xls/app/video/rpc/video/internal/model"

	"xls/app/video/rpc/video/internal/code"
	"xls/app/video/rpc/video/internal/svc"
	"xls/app/video/rpc/video/video"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishLogic {
	return &PublishLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PublishLogic) Publish(in *video.PublishRequest) (*video.PublishResponse, error) {
	resp := new(video.PublishResponse)

	db := l.svcCtx.MysqlDB

	// 处理 tags
	var tags []model.Tag

	for _, tagName := range in.Tags {
		if tagName == "" {
			continue
		}

		var tag model.Tag

		// 查找 tag 是否存在
		err := db.Where("name = ?", tagName).First(&tag).Error

		if err != nil {
			// 不存在则创建
			if errors.Is(err, gorm.ErrRecordNotFound) {
				tag = model.Tag{
					Name: tagName,
				}

				if err := db.Create(&tag).Error; err != nil {
					l.Logger.Errorf("create tag error: %v", err)
					resp.Error = code.FAILED
					return resp, nil
				}
			} else {
				l.Logger.Errorf("query tag error: %v", err)
				resp.Error = code.FAILED
				return resp, nil
			}
		}

		tags = append(tags, tag)
	}

	// 创建视频
	newVideo := &model.Video{
		Uid:   uint(in.Uid),
		Title: in.Title,
		Url:   in.Url,
		Tags:  tags,
	}

	if err := db.Create(newVideo).Error; err != nil {
		l.Logger.Errorf("insert video to mysql error: %v", err)
		resp.Error = code.FAILED
		return resp, nil
	}

	resp.VideoID = int32(newVideo.ID)
	resp.Error = code.SUCCEED

	return resp, nil
}
