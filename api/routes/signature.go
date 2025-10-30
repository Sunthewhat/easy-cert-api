package routes

import (
	"github.com/gofiber/fiber/v2"
	signature_controller "github.com/sunthewhat/easy-cert-api/api/controllers/signature"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
)

func SetupSignatureRoutes(router fiber.Router) {
	signatureGroup := router.Group("signature")

	signatureGroup.Use(middleware.AuthMiddleware())

	signatureGroup.Post("", signature_controller.Create)
	signatureGroup.Get("resign/:signatureId", signature_controller.RequestResign)
	signatureGroup.Get("signer/:certificateId", signature_controller.GetSignerData)
	signatureGroup.Get(":id", signature_controller.GetById)
	signatureGroup.Put("sign/:id", signature_controller.Sign)
	signatureGroup.Get(":certificateId/:signerId", signature_controller.GetSignatureImage)
}
