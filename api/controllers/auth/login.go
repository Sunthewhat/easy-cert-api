package auth_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/model/userModel"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Login(c *fiber.Ctx) error {
	body := new(payload.LoginPayload)

	if err := c.BodyParser(body); err != nil {
		return response.SendError(c, "Failed to parse body")
	}

	if err := util.ValidateStruct(body); err != nil {
		errors := util.GetValidationErrors(err)
		return response.SendFailed(c, errors[0]) // Return first validation error
	}

	user, queryErr := userModel.GetByUsername(body.Username)

	if user == nil {
		if queryErr != nil {
			slog.Error("Auth Login database query failed", "error", queryErr, "username", body.Username)
			return response.SendInternalError(c, queryErr)
		} else {
			slog.Info("Auth Login attempt with non-existent user", "username", body.Username)
			return response.SendFailed(c, "User not found")
		}
	}

	if isPasswordMatch := util.CheckPassword(body.Password, user.Password); !isPasswordMatch {
		slog.Warn("Auth Login failed password check", "username", body.Username)
		return response.SendFailed(c, "Incorrect Password")
	}

	authToken, err := util.GenerateAuthToken(user.ID)

	if err != nil {
		slog.Error("Auth Login JWT generation failed", "error", err, "user_id", user.ID)
		return response.SendError(c, "Failed to generate JWT Token")
	}

	slog.Info("Auth Login successful", "username", body.Username, "user_id", user.ID)
	return response.SendSuccess(c, "Login Successfully", fiber.Map{"token": authToken})
}
