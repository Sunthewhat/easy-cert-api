package auth_controller

import (
	"github.com/bsthun/gut"
	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/secure-docs-api/api/model/userModel"
	"github.com/sunthewhat/secure-docs-api/common/util"
	"github.com/sunthewhat/secure-docs-api/type/payload"
	"github.com/sunthewhat/secure-docs-api/type/response"
)

func Register(c *fiber.Ctx) error {
	body := new(payload.RegisterPayload)

	// Parse Body to struct
	if err := c.BodyParser(body); err != nil {
		return response.SendError(c, "Failed to parse body")
	}

	// Validate Body structure
	if validateErr := gut.Validate(body); validateErr != nil {
		return response.SendFailed(c, "Missing required fields")
	}

	// Check if username already existed
	if dupUser, err := userModel.GetByUsername(body.Username); dupUser != nil || err != nil {
		if dupUser != nil {
			return response.SendFailed(c, "User already existed")
		}
		return response.SendInternalError(c, err)
	}

	// Hasing Password
	hashedPassword, hashErr := util.HashPassword(body.Password)

	if hashErr != nil {
		return response.SendError(c, "Password hashing failed")
	}

	createdUser, createErr := userModel.CreateNewUser(body.Username, hashedPassword, body.Firstname, body.Lastname)

	if createErr != nil {
		return response.SendError(c, "Failed to create user")
	}

	return response.SendSuccess(c, "User Registered", fiber.Map{
		"id":       createdUser.ID,
		"username": createdUser.Username,
	})
}
