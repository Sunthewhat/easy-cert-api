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
	certificateGroup.Get(":certId", certificate_controller.GetById)
	certificateGroup.Post("", certificate_controller.Create)
	certificateGroup.Put(":id", certificate_controller.Update)
	certificateGroup.Delete(":certId", certificate_controller.Delete)
	certificateGroup.Post("render/:certId", certificate_controller.Render)
	certificateGroup.Get("mail/:certId", certificate_controller.DistributeByMail)
	certificateGroup.Post("mail/resend/:participantId", certificate_controller.ResendParticipantMail)
	certificateGroup.Get("anchor/:certId", certificate_controller.GetAnchorList)
	certificateGroup.Get("generate/status/:certificateId", certificate_controller.CheckGenerateStatus)
}
