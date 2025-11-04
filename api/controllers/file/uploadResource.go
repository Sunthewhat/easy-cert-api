package file

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func UploadResource(c *fiber.Ctx) error {
	resourceType := c.Params("type")

	if resourceType != "background" && resourceType != "graphic" {
		return response.SendFailed(c, "Invalid resource type")
	}

	// Get user ID from context (set by AuthMiddleware)
	userId, ok := middleware.GetUserFromContext(c)
	if !ok {
		return response.SendUnauthorized(c, "User not authenticated")
	}

	file, err := c.FormFile("image")

	if err != nil {
		return response.SendFailed(c, "No file provided")
	}

	if file.Size > 15*1024*1024 {
		return response.SendFailed(c, fmt.Sprintf("File size too large (%dMB out off 15MB)", file.Size/(1024*1024)))
	}

	ext := filepath.Ext(file.Filename)
	uniqueID := uuid.New().String()
	timeStamp := time.Now().Unix()

	// Include userId in the object path: userId/resourceType_timestamp_uuid.ext
	objName := fmt.Sprintf("%s/%s_%d_%s%s", userId, resourceType, timeStamp, uniqueID, ext)

	ctx := context.Background()

	fileURL, err := util.UploadFile(ctx, *common.Config.BucketResource, objName, file)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	// Convert MinIO URL to backend proxy URL for security
	proxyURL, err := util.ConvertToProxyURL(fileURL, *common.Config.BucketResource)
	if err != nil {
		// If conversion fails, use original URL as fallback
		proxyURL = fileURL
	}

	return response.SendSuccess(c, "Resource Upload Successfully", fiber.Map{
		"filename":    file.Filename,
		"object_name": objName,
		"url":         proxyURL,
		"size":        file.Size,
	})
}
