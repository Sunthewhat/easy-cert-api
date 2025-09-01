package middleware

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

// AuthMiddleware - Complete JWT authentication middleware
func AuthMiddleware() fiber.Handler {
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

		newToken, err := util.RefreshSSO(token)
		if err != nil {
			slog.Error("Refresh token failed", "error", err)
			return response.SendUnauthorized(c, err.Error())
		}

		jwtPayload, err := util.DecodeJWTToken(newToken.AccessToken)
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

// OptionalAuthMiddleware - JWT authentication middleware that doesn't block unauthenticated requests
// Sets user context if valid token is provided, but allows requests to continue without authentication
func OptionalAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")

		// No auth header - continue as unauthenticated user
		if authHeader == "" {
			c.Locals("is_authenticated", false)
			slog.Debug("OptionalAuthMiddleware: no auth header, continuing as unauthenticated",
				"path", c.Path(),
				"method", c.Method())
			return c.Next()
		}

		// Try to parse token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Locals("is_authenticated", false)
			slog.Debug("OptionalAuthMiddleware: invalid auth header format, continuing as unauthenticated",
				"path", c.Path(),
				"method", c.Method())
			return c.Next()
		}

		token := tokenParts[1]
		claims, err := util.DecodeAuthToken(token)
		if err != nil {
			c.Locals("is_authenticated", false)
			slog.Debug("OptionalAuthMiddleware: invalid token, continuing as unauthenticated",
				"error", err,
				"path", c.Path(),
				"method", c.Method())
			return c.Next()
		}

		// Valid token - set user context
		if claims.UserId != nil && *claims.UserId != "" {
			c.Locals("user_id", *claims.UserId)
			c.Locals("jwt_claims", claims)
			c.Locals("is_authenticated", true)

			slog.Info("OptionalAuthMiddleware: authentication successful",
				"user_id", *claims.UserId,
				"path", c.Path(),
				"method", c.Method())
		} else {
			c.Locals("is_authenticated", false)
		}

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

// IsAuthenticated - Helper function to check if user is authenticated
func IsAuthenticated(c *fiber.Ctx) bool {
	if auth := c.Locals("is_authenticated"); auth != nil {
		if isAuth, ok := auth.(bool); ok {
			return isAuth
		}
	}
	return false
}
