package util

import (
	"context"
	"fmt"
	"mime/multipart"
	"strings"

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
	common.MinIOClient = client
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

func DownloadFile(ctx context.Context, bucketName string, objectName string) (*minio.Object, error) {
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

func ListFilesByPrefix(ctx context.Context, bucketName string, prefix string, limit int) ([]string, error) {
	if minioClient == nil {
		return nil, fmt.Errorf("MinIO client not initialized")
	}

	var fileURLs []string
	count := 0

	objectCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}

		if limit > 0 && count >= limit {
			break
		}

		url := fmt.Sprintf("https://%s/%s/%s", *common.Config.MinIoEndpoint, bucketName, object.Key)
		fileURLs = append(fileURLs, url)
		count++
	}

	return fileURLs, nil
}

// ExtractObjectNameFromURL extracts the object name from a MinIO URL
// Example: https://endpoint/bucket/path/to/file.pdf -> path/to/file.pdf
func ExtractObjectNameFromURL(url string, bucketName string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("URL is empty")
	}

	// Find the bucket name in the URL and extract everything after it
	bucketPrefix := fmt.Sprintf("/%s/", bucketName)
	idx := strings.Index(url, bucketPrefix)
	if idx == -1 {
		return "", fmt.Errorf("bucket name not found in URL")
	}

	objectName := url[idx+len(bucketPrefix):]
	if objectName == "" {
		return "", fmt.Errorf("object name is empty")
	}

	return objectName, nil
}

// DeleteFileByURL deletes a file from MinIO given its full URL
func DeleteFileByURL(ctx context.Context, bucketName string, fileURL string) error {
	if fileURL == "" {
		// If URL is empty, nothing to delete
		return nil
	}

	objectName, err := ExtractObjectNameFromURL(fileURL, bucketName)
	if err != nil {
		return fmt.Errorf("failed to extract object name from URL: %w", err)
	}

	return DeleteFile(ctx, bucketName, objectName)
}

// ConvertToProxyURL converts a direct MinIO URL to a backend proxy URL
// Example: https://minio.example.com/bucket/path/file.pdf -> http://localhost:8000/api/public/files/download/bucket/path/file.pdf
func ConvertToProxyURL(minioURL string, bucketName string) (string, error) {
	if minioURL == "" {
		return "", nil
	}

	objectName, err := ExtractObjectNameFromURL(minioURL, bucketName)
	if err != nil {
		return "", fmt.Errorf("failed to extract object name from URL: %w", err)
	}

	return fmt.Sprintf("%s/api/public/files/download/%s/%s", *common.Config.BackendURL, bucketName, objectName), nil
}

// GenerateProxyURL generates a backend proxy URL for a given bucket and object path
func GenerateProxyURL(bucketName string, objectPath string) string {
	return fmt.Sprintf("%s/api/public/files/download/%s/%s", *common.Config.BackendURL, bucketName, objectPath)
}