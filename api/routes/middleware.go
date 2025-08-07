package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/secure-docs-api/type/response"
)

// AuthMiddleware - JWT authentication middleware
// This is a placeholder - implement your actual JWT validation logic
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO: Implement JWT token validation
		// For now, we'll set a dummy user_id for testing
		// Replace this with actual JWT parsing and validation

		token := c.Get("Authorization")
		if token == "" {
			return response.SendUnauthorized(c, "Authorization header required")
		}

		// TODO: Parse and validate JWT token
		// Extract user information from token
		// Set user context

		// For testing purposes, set a dummy user_id
		c.Locals("user_id", "test_user_123")
		c.Locals("user_role", "user")

		return c.Next()
	}
}

// AdminMiddleware - Admin role validation middleware
func AdminMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("user_role")
		if userRole != "admin" {
			return response.SendUnauthorized(c, "Admin access required")
		}

		return c.Next()
	}
}

// OptionalAuthMiddleware - Optional authentication for endpoints that work with or without auth
func OptionalAuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token != "" {
			// TODO: Parse and validate JWT token if present
			// Set user context if valid
			c.Locals("user_id", "test_user_123")
			c.Locals("user_role", "user")
		}

		return c.Next()
	}
}
