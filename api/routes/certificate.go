package routes

import (
	"github.com/gofiber/fiber/v2"
	certificate_controller "github.com/sunthewhat/easy-cert-api/api/controllers/certificate"
)

func SetupCertificateRoutes(router fiber.Router) {
	certificateGroup := router.Group("certificate")

	certificateGroup.Get("", certificate_controller.GetAll)
	certificateGroup.Put(":id", certificate_controller.Update)
	certificateGroup.Delete(":certId", certificate_controller.Delete)
}
