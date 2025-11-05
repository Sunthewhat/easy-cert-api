package middleware

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

// AuthMiddleware - Complete JWT authentication middleware
func AuthMiddleware(ssoService util.ISSOService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			slog.Warn("AuthMiddleware: missing authorization header",
				"path", c.Path(),
				"method", c.Method(),
				"ip", c.IP())
			return response.SendUnauthorized(c, "Authorization header is required")
		}

		// Extract token from "Bearer <token>" format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			slog.Warn("AuthMiddleware: invalid authorization header format",
				"path", c.Path(),
				"method", c.Method(),
				"header", authHeader,
				"ip", c.IP())
			return response.SendUnauthorized(c, "Invalid authorization header format. Expected: Bearer <token>")
		}

		token := tokenParts[1]

		newToken, err := ssoService.Refresh(token)
		if err != nil {
			slog.Error("Refresh token failed", "error", err)
			return response.SendUnauthorized(c, err.Error())
		}

		jwtPayload, err := ssoService.Decode(newToken.AccessToken)
		if err != nil {
			slog.Error("Failed to decode JWT token from refreshed token", "errror", err)
		}

		// Set user information in context for use by handlers
		c.Locals("user_id", jwtPayload.Email)
		// c.Locals("refresh_token", newToken.RefreshToken)
		c.Set("X-Refresh-Token", newToken.RefreshToken)

		slog.Info("AuthMiddleware: authentication successful",
			"user_id", jwtPayload.Email,
			"path", c.Path(),
			"method", c.Method(),
			"ip", c.IP())

		// Continue to next handler
		return c.Next()
	}
}

// GetUserFromContext - Helper function to extract user ID from request context
func GetUserFromContext(c *fiber.Ctx) (string, bool) {
	if userID := c.Locals("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			return id, true
		}
	}
	return "", false
}
