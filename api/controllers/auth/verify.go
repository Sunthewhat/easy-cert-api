package auth_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Verify(c *fiber.Ctx) error {
	userId, success := middleware.GetUserFromContext(c)
	if !success {
		slog.Error("Get user from context failed")
		return response.SendUnauthorized(c, "Failed to read user from context")
	}
	slog.Info("Auth Verify successful", "user_id", userId)
	// Return success with user claims
	return response.SendSuccess(c, "Token is valid", map[string]any{
		"userId": userId,
	})
}
