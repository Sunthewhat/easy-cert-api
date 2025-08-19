package auth_controller

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Verify(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	if authHeader == "" {
		slog.Warn("Auth Verify attempt without authorization header")
		return response.SendFailed(c, "Authorization header not found")
	}

	// Extract token from "Bearer <token>" format
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		slog.Warn("Auth Verify attempt with invalid header format")
		return response.SendFailed(c, "Invalid authorization header format")
	}

	token := tokenParts[1]

	// Decode and validate the token
	claims, err := util.DecodeAuthToken(token)
	if err != nil {
		slog.Warn("Auth Verify attempt with invalid/expired token", "error", err)
		return response.SendFailed(c, "Invalid or expired token")
	}

	slog.Info("Auth Verify successful", "user_id", claims.UserId)
	// Return success with user claims
	return response.SendSuccess(c, "Token is valid", map[string]any{
		"userId": claims.UserId,
		"exp":    claims.ExpiresAt,
		"iat":    claims.IssuedAt,
	})
}
