package routes

import (
	"github.com/gofiber/fiber/v2"
	auth_controller "github.com/sunthewhat/easy-cert-api/api/controllers/auth"
)

func SetupAuthRoutes(router fiber.Router) {
	authGroup := router.Group("auth")

	authGroup.Post("login", auth_controller.Login)
	authGroup.Post("register", auth_controller.Register)
}
