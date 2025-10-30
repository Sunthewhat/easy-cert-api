package routes

import (
	"github.com/gofiber/fiber/v2"
	signer_controller "github.com/sunthewhat/easy-cert-api/api/controllers/signer"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
)

func SetupSignerRoutes(router fiber.Router) {
	signerGroup := router.Group("signer")

	signerGroup.Use(middleware.AuthMiddleware())

	signerGroup.Get("", signer_controller.GetByUser)
	signerGroup.Post("", signer_controller.Create)
	signerGroup.Get("status/:certId", signer_controller.GetStatus)
}
