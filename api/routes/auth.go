package routes

import (
	"github.com/gofiber/fiber/v2"
	auth_controller "github.com/sunthewhat/easy-cert-api/api/controllers/auth"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
)

func SetupAuthRoutes(router fiber.Router) {
	var authService util.IAuthService

	// Choose authentication service based on config
	if *common.Config.IsSsoEnable {
		authService = util.NewSSOService()
	} else {
		authService = util.NewInternalAuthService()
	}

	authCtrl := auth_controller.NewAuthController(authService)
	authGroup := router.Group("auth")

	authGroup.Post("login", authCtrl.Login)
	authGroup.Get("verify", middleware.AuthMiddleware(authService), authCtrl.Verify)
}
