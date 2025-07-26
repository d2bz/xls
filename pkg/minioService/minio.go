package minioService

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinioClient(endpoint, accessKey, secretKey, bucket, baseUrl string, useSSL bool) (*minio.Client, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	// 创建桶（如果不存在）
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		err = minioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return nil, err
		}
	}

	return minioClient, nil
}

// 上传文件
// func (m *MinioClient) UploadFile(ctx context.Context, objectName, filePath, contentType string) (string, error) {
// 	info, err := m.Client.FPutObject(ctx, m.BucketName, objectName, filePath, minio.PutObjectOptions{
// 		ContentType: contentType,
// 	})
// 	if err != nil {
// 		return "", err
// 	}
// 	log.Printf("Uploaded %s of size %d\n", objectName, info.Size)
// 	return fmt.Sprintf("%s/%s/%s", m.BaseUrl, m.BucketName, objectName), nil
// }
