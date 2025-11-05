package routes

import (
	"github.com/gofiber/fiber/v2"
	auth_controller "github.com/sunthewhat/easy-cert-api/api/controllers/auth"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	"github.com/sunthewhat/easy-cert-api/common/util"
)

func SetupAuthRoutes(router fiber.Router) {
	ssoService := util.NewSSOService()
	authCtrl := auth_controller.NewAuthController(ssoService)
	authGroup := router.Group("auth")

	authGroup.Post("login", authCtrl.Login)
	authGroup.Get("verify", middleware.AuthMiddleware(ssoService), authCtrl.Verify)
}
