package routes

import (
	"github.com/gofiber/fiber/v2"
	certificate_controller "github.com/sunthewhat/easy-cert-api/api/controllers/certificate"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	"github.com/sunthewhat/easy-cert-api/common"
)

func SetupCertificateRoutes(router fiber.Router) {
	// Initialize repositories
	certRepo := certificatemodel.NewCertificateRepository(common.Gorm)
	signatureRepo := signaturemodel.NewSignatureRepository(common.Gorm)

	// Initialize certificate controller with dependencies
	certCtrl := certificate_controller.NewCertificateController(certRepo, signatureRepo)

	certificateGroup := router.Group("certificate")

	certificateGroup.Use(middleware.AuthMiddleware())

	certificateGroup.Get("", certCtrl.GetByUser)
	certificateGroup.Get(":certId", certCtrl.GetById)
	certificateGroup.Post("", certCtrl.Create)
	certificateGroup.Put(":id", certCtrl.Update)
	certificateGroup.Delete(":certId", certCtrl.Delete)
	certificateGroup.Post("render/:certId", certCtrl.Render)
	certificateGroup.Get("mail/:certId", certCtrl.DistributeByMail)
	certificateGroup.Post("mail/resend/:participantId", certificate_controller.ResendParticipantMail)
	certificateGroup.Get("anchor/:certId", certCtrl.GetAnchorList)
	certificateGroup.Get("generate/status/:certificateId", certCtrl.CheckGenerateStatus)
}
