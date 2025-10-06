package routes

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sunthewhat/easy-cert-api/api/handler"
	"github.com/sunthewhat/easy-cert-api/common"
)

// Init initializes all routes and middleware
func Init(app *fiber.App) {
	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New())

	// Configure CORS with origins from config
	var allowedOrigins string
	if len(common.Config.Cors) > 0 {
		// Convert []*string to []string
		origins := make([]string, len(common.Config.Cors))
		for i, origin := range common.Config.Cors {
			if origin != nil {
				origins[i] = *origin
			}
		}
		allowedOrigins = strings.Join(origins, ",")
	} else {
		allowedOrigins = "*" // Fallback to wildcard if no config
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	// API routes
	api := app.Group("/api")

	// Health check endpoint
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "EasyCert API is running",
		})
	})

	// Handle OPTIONS requests (CORS preflight)
	api.Options("/health", func(c *fiber.Ctx) error {
		slog.Debug("Health Check OPTIONS request",
			"method", c.Method(),
			"path", c.Path(),
			"user_agent", c.Get("User-Agent"))
		return c.SendStatus(200)
	})

	// Public routes
	SetupPublicRoutes(api)

	// API versioning
	v1 := api.Group("/v1")

	// Setup all route modules
	SetupAuthRoutes(v1)
	SetupCertificateRoutes(v1)
	SetupParticipantRoutes(v1)
	SetupFileRoutes(v1)
	SetupSignerRoutes(v1)
	SetupSignatureRoutes(v1)

	// Handle favicon requests to prevent 404s
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendStatus(204) // No Content
	})

	// 404 handler
	app.Use(handler.HandleNotFound)
}

// SetupPublicRoutes configures public routes
func SetupPublicRoutes(router fiber.Router) {
	publicGroup := router.Group("/public")

	// Public endpoints for health checks, documentation, etc.
	publicGroup.Get("/status", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"timestamp": fiber.Map{},
		})
	})
}
