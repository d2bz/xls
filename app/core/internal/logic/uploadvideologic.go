package logic

import (
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"time"

	"xls/app/core/internal/code"
	"xls/app/core/internal/svc"
	"xls/app/core/internal/types"
	"xls/pkg/minioService"

	"github.com/minio/minio-go/v7"
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

func (l *UploadVideoLogic) UploadVideo(file *multipart.File, header *multipart.FileHeader) (resp *types.UploadVideoResponse, err error) {
	resp = new(types.UploadVideoResponse)

	minioCli, err := minioService.NewMinioClient(
		l.svcCtx.Config.Minio.Endpoint,
		l.svcCtx.Config.Minio.AccessKey,
		l.svcCtx.Config.Minio.SecretKey,
		l.svcCtx.Config.Minio.Bucket,
		l.svcCtx.Config.Minio.BaseUrl,
		l.svcCtx.Config.Minio.UseSSL,
	)
	if err != nil {
		resp.Status = code.FAILED
		l.Logger.Errorf("create miniocli error: %v", err)
		return
	}
	objectName := time.Now().Format("20060102_150405") + filepath.Ext(header.Filename)
	contentType := header.Header.Get("Content-Type") //得到文件的类型
	_, err = minioCli.PutObject(context.Background(), l.svcCtx.Config.Minio.Bucket, objectName, *file, header.Size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		resp.Status = code.FAILED
		l.Logger.Errorf("upload file to bucket error: %v", err)
		return
	}

	// 构建文件的 URL
	fileURL := fmt.Sprintf("http://%s/%s/%s", l.svcCtx.Config.Minio.Endpoint, l.svcCtx.Config.Minio.Bucket, objectName)

	resp = &types.UploadVideoResponse{
		Url: fileURL,
	}

	return resp, nil
}

// func (l *UploadVideoLogic) UploadSmallVideo(file []byte, fileName string) (resp *types.UploadVideoResponse, err error) {

// 	return
// }
