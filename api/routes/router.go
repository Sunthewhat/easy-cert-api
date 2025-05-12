package routes

import (
	"github.com/gofiber/fiber/v2"
	auth_controller "github.com/sunthewhat/secure-docs-api/api/controllers/auth"
)

func Init(router fiber.Router) {
	api := router.Group("api")

	publicGroup := api.Group("public")

	authGroup := publicGroup.Group("auth")

	authGroup.Post("register", auth_controller.Register)
	authGroup.Post("login", auth_controller.Login)
}
