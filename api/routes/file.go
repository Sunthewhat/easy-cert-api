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
