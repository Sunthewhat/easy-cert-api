package auth_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
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

	ssoResponse, err := util.LoginSSO(body.Username, body.Password)

	if err != nil {
		slog.Error("SSO login Failed")
		return response.SendInternalError(c, err)
	}

	// Decode the JWT access token
	jwtPayload, err := util.DecodeJWTToken(ssoResponse.AccessToken)
	if err != nil {
		slog.Error("Failed to decode JWT token", "error", err)
		return response.SendInternalError(c, err)
	}

	return response.SendSuccess(c, "Login Successfull", fiber.Map{
		"token":     ssoResponse.RefreshToken,
		"firstname": jwtPayload.GivenName,
		"lastname":  jwtPayload.FamilyName,
		"username":  jwtPayload.PreferredUsername,
	})
}
