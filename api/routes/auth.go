package routes

import (
	"github.com/gofiber/fiber/v2"
	auth_controller "github.com/sunthewhat/easy-cert-api/api/controllers/auth"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
)

func SetupAuthRoutes(router fiber.Router) {
	authGroup := router.Group("auth")

	authGroup.Post("login", auth_controller.Login)
	authGroup.Get("verify", middleware.AuthMiddleware(), auth_controller.Verify)
}
