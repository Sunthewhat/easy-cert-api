package file

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func GetAllResourceByType(c *fiber.Ctx) error {
	resourceType := c.Params("type")

	if resourceType != "background" && resourceType != "graphic" {
		return response.SendFailed(c, "Invalid resource type")
	}

	ctx := context.Background()

	// Get direct MinIO URLs first
	minioURLs, err := util.ListFilesByPrefix(ctx, *common.Config.BucketResource, resourceType, 0)
	if err != nil {
		return response.SendInternalError(c, err)
	}

	// Convert MinIO URLs to backend proxy URLs for security
	proxyURLs := make([]string, len(minioURLs))
	for i, minioURL := range minioURLs {
		proxyURL, err := util.ConvertToProxyURL(minioURL, *common.Config.BucketResource)
		if err != nil {
			// If conversion fails, log and use original URL as fallback
			proxyURLs[i] = minioURL
		} else {
			proxyURLs[i] = proxyURL
		}
	}

	return response.SendSuccess(c, "Resources retrieved successfully", fiber.Map{
		"type":  resourceType,
		"count": len(proxyURLs),
		"files": proxyURLs,
	})
}
