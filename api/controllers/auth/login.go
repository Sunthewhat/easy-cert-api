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
		return response.SendError(c, "Failed to parse body")
	}

	if validateErr := gut.Validate(body); validateErr != nil {
		return response.SendFailed(c, "Missing required fields")
	}

	user, queryErr := userModel.GetByUsername(body.Username)

	if user == nil {
		if queryErr != nil {
			return response.SendInternalError(c, queryErr)
		} else {
			return response.SendFailed(c, "User not found")
		}
	}

	if isPasswordMatch := util.CheckPassword(body.Password, user.Password); !isPasswordMatch {
		return response.SendFailed(c, "Incorrect Password")
	}

	authToken, err := util.GenerateAuthToken(user.ID)

	if err != nil {
		return response.SendError(c, "Failed to generate JWT Token")
	}

	return response.SendSuccess(c, "Login Successfully", fiber.Map{"token": authToken})
}
