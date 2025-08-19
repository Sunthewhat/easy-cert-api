package auth_controller

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/model/userModel"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Register(c *fiber.Ctx) error {
	body := new(payload.RegisterPayload)

	// Parse Body to struct
	if err := c.BodyParser(body); err != nil {
		return response.SendError(c, "Failed to parse body")
	}

	// Validate Body structure
	if err := util.ValidateStruct(body); err != nil {
		errors := util.GetValidationErrors(err)
		return response.SendFailed(c, errors[0]) // Return first validation error
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
