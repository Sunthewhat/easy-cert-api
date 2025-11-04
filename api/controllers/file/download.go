package file

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

// DownloadFile serves files from MinIO through the backend as a secure proxy
func DownloadFile(c *fiber.Ctx) error {
	// Get bucket and object path from URL params
	bucket := c.Params("bucket")
	objectPath := c.Params("*")

	if bucket == "" || objectPath == "" {
		slog.Warn("File download attempt with missing parameters", "bucket", bucket, "objectPath", objectPath)
		return response.SendFailed(c, "Invalid file path")
	}

	// Validate bucket - only allow specific buckets
	validBuckets := map[string]bool{
		*common.Config.BucketCertificate: true,
		*common.Config.BucketResource:    true,
	}

	if !validBuckets[bucket] {
		slog.Warn("File download attempt with invalid bucket", "bucket", bucket)
		return response.SendFailed(c, "Invalid bucket")
	}

	ctx := context.Background()

	// Download file from MinIO
	object, err := util.DownloadFile(ctx, bucket, objectPath)
	if err != nil {
		slog.Error("File download failed", "error", err, "bucket", bucket, "objectPath", objectPath)
		return response.SendError(c, "File not found")
	}
	defer object.Close()

	// Read the object stats to get content type and size
	objectInfo, err := object.Stat()
	if err != nil {
		slog.Error("Failed to get file stats", "error", err, "bucket", bucket, "objectPath", objectPath)
		return response.SendInternalError(c, err)
	}

	// Determine content type based on file extension
	contentType := "application/octet-stream"
	if strings.HasSuffix(objectPath, ".pdf") {
		contentType = "application/pdf"
	} else if strings.HasSuffix(objectPath, ".zip") {
		contentType = "application/zip"
	} else if strings.HasSuffix(objectPath, ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(objectPath, ".jpg") || strings.HasSuffix(objectPath, ".jpeg") {
		contentType = "image/jpeg"
	} else if strings.HasSuffix(objectPath, ".svg") {
		contentType = "image/svg+xml"
	}

	// Set response headers
	c.Set("Content-Type", contentType)
	c.Set("Content-Length", fmt.Sprintf("%d", objectInfo.Size))
	c.Set("Content-Disposition", "inline") // Display in browser instead of forcing download

	// Stream the file to the response
	_, err = io.Copy(c.Response().BodyWriter(), object)
	if err != nil {
		slog.Error("Failed to stream file", "error", err, "bucket", bucket, "objectPath", objectPath)
		return response.SendInternalError(c, err)
	}

	slog.Info("File downloaded successfully", "bucket", bucket, "objectPath", objectPath, "size", objectInfo.Size)
	return nil
}
