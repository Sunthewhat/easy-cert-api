package file

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func DeleteResource(c *fiber.Ctx) error {
	resourceType := c.Params("type")

	if resourceType != "background" && resourceType != "graphic" {
		return response.SendFailed(c, "Invalid resource type")
	}

	// Get the file URL or object name from request body
	type DeleteRequest struct {
		URL        string `json:"url"`
		ObjectName string `json:"object_name"`
	}

	var req DeleteRequest
	if err := c.BodyParser(&req); err != nil {
		return response.SendFailed(c, "Invalid request body")
	}

	if req.URL == "" && req.ObjectName == "" {
		return response.SendFailed(c, "Either 'url' or 'object_name' must be provided")
	}

	ctx := context.Background()

	var err error
	var objectName string

	// Delete by URL or object name
	if req.URL != "" {
		// Extract object name from URL
		objectName, err = util.ExtractObjectNameFromURL(req.URL, *common.Config.BucketResource)
		if err != nil {
			return response.SendFailed(c, "Invalid URL format")
		}
	} else {
		objectName = req.ObjectName
	}

	// Validate that the object name starts with the correct resource type
	if !strings.HasPrefix(objectName, resourceType) {
		return response.SendFailed(c, "Object does not match the specified resource type")
	}

	// Delete the file
	err = util.DeleteFile(ctx, *common.Config.BucketResource, objectName)
	if err != nil {
		return response.SendInternalError(c, err)
	}

	return response.SendSuccess(c, "Resource deleted successfully", fiber.Map{
		"object_name": objectName,
		"type":        resourceType,
	})
}
