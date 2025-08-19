package auth_controller

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Verify(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	if authHeader == "" {
		return response.SendFailed(c, "Authorization header not found")
	}

	// Extract token from "Bearer <token>" format
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return response.SendFailed(c, "Invalid authorization header format")
	}

	token := tokenParts[1]

	// Decode and validate the token
	claims, err := util.DecodeAuthToken(token)
	if err != nil {
		return response.SendFailed(c, "Invalid or expired token")
	}

	// Return success with user claims
	return response.SendSuccess(c, "Token is valid", map[string]any{
		"userId": claims.UserId,
		"exp":    claims.ExpiresAt,
		"iat":    claims.IssuedAt,
	})
}
