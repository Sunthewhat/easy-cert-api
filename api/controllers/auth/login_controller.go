package auth_controller

import (
	"github.com/bsthun/gut"
	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/secure-docs-api/api/model/userModel"
	"github.com/sunthewhat/secure-docs-api/common/util"
	"github.com/sunthewhat/secure-docs-api/type/payload"
	"github.com/sunthewhat/secure-docs-api/type/response"
)

func Login(c *fiber.Ctx) error {
	body := new(payload.LoginPayload)

	if err := c.BodyParser(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error(response.Error("Failed to parse body")))
	}

	if validateErr := gut.Validate(body); validateErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("Missing required fields"))
	}

	user, queryErr := userModel.GetByUsername(body.Username)

	if user == nil {
		if queryErr != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(response.Error(queryErr.Error()))
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(response.Error("User not found"))
		}
	}

	if isPasswordMatch := util.CheckPassword(body.Password, user.Password); !isPasswordMatch {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error("Incorrect Password"))
	}

	authToken, err := util.GenerateAuthToken(int(user.ID))

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error("Failed to generate JWT token"))
	}

	return c.Status(fiber.StatusOK).JSON(response.Success("Login Successfully", fiber.Map{
		"token": authToken,
	}))
}
