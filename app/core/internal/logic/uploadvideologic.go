package logic

import (
	"context"
	"mime/multipart"

	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadVideoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadVideoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadVideoLogic {
	return &UploadVideoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UploadVideoLogic) UploadLargeVideo(file *multipart.File, fileName string) (resp *types.UploadVideoResponse, err error) {

	return
}

func (l *UploadVideoLogic) UploadSmallVideo(file []byte, fileName string) (resp *types.UploadVideoResponse, err error) {

	return
}
