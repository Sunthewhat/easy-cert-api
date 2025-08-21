package routes

import (
	"github.com/gofiber/fiber/v2"
	certificate_controller "github.com/sunthewhat/easy-cert-api/api/controllers/certificate"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
)

func SetupCertificateRoutes(router fiber.Router) {
	certificateGroup := router.Group("certificate")

	certificateGroup.Use(middleware.AuthMiddleware())

	certificateGroup.Get("", certificate_controller.GetByUser)
	certificateGroup.Post("", certificate_controller.Create)
	certificateGroup.Put(":id", certificate_controller.Update)
	certificateGroup.Delete(":certId", certificate_controller.Delete)
	certificateGroup.Post("render/:certId", certificate_controller.Render)
}
