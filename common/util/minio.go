package util

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sunthewhat/easy-cert-api/common"
)

var minioClient *minio.Client

func InitMinIO() error {
	if common.Config.MinIoEndpoint == nil || common.Config.MinIoAccessKey == nil || common.Config.MinIoSecretKey == nil {
		return fmt.Errorf("MinIO configuration is incomplete")
	}

	client, err := minio.New(*common.Config.MinIoEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(*common.Config.MinIoAccessKey, *common.Config.MinIoSecretKey, ""),
		Secure: true,
	})

	if err != nil {
		return fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	minioClient = client
	return nil
}

func UploadFile(ctx context.Context, bucketName string, objectName string, file *multipart.FileHeader) (string, error) {
	if minioClient == nil {
		return "", fmt.Errorf("MinIO client not initialized")
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Check if bucket exists, if not create it
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return "", fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	// Upload the file
	info, err := minioClient.PutObject(ctx, bucketName, objectName, src, file.Size, minio.PutObjectOptions{
		ContentType: file.Header.Get("Content-Type"),
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Return the object URL or info
	url := fmt.Sprintf("https://%s/%s/%s", *common.Config.MinIoEndpoint, bucketName, objectName)
	_ = info // We have the upload info if needed

	return url, nil
}

func DownloadFile(ctx context.Context, bucketName string, objectName string) (io.ReadCloser, error) {
	if minioClient == nil {
		return nil, fmt.Errorf("MinIO client not initialized")
	}

	object, err := minioClient.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	return object, nil
}

func DeleteFile(ctx context.Context, bucketName string, objectName string) error {
	if minioClient == nil {
		return fmt.Errorf("MinIO client not initialized")
	}

	err := minioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}