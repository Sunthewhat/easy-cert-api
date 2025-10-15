package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/controllers/file"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
)

func SetupFileRoutes(app fiber.Router) {
	fileGroup := app.Group("/files")

	// Apply JWT middleware to protect file operations
	fileGroup.Use(middleware.AuthMiddleware())

	// File upload endpoint
	fileGroup.Post("/:type", file.UploadResource)

	// Get all files by type endpoint
	fileGroup.Get("/:type", file.GetAllResourceByType)
}

// SetupPublicFileRoutes configures public file download routes
func SetupPublicFileRoutes(app fiber.Router) {
	// Public file download endpoint - serves files without authentication
	// This is needed for direct browser access (img tags, a tags, etc.)
	// Security is handled by the controller (validates file access)
	app.Get("/files/download/:bucket/*", file.DownloadFile)

	// Public certificate download endpoint for participants
	// This validates the participant ID before serving the file
	app.Get("/certificate/:participantId", file.PublicDownloadCertificate)
}
