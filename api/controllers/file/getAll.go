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

	fileURLs, err := util.ListFilesByPrefix(ctx, *common.Config.BucketResource, resourceType, 0)
	if err != nil {
		return response.SendInternalError(c, err)
	}

	return response.SendSuccess(c, "Resources retrieved successfully", fiber.Map{
		"type":  resourceType,
		"count": len(fileURLs),
		"files": fileURLs,
	})
}
