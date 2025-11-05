package routes

import (
	"github.com/gofiber/fiber/v2"
	certificate_controller "github.com/sunthewhat/easy-cert-api/api/controllers/certificate"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
)

func SetupCertificateRoutes(router fiber.Router) {
	// Initialize repositories
	certRepo := certificatemodel.NewCertificateRepository(common.Gorm)
	signatureRepo := signaturemodel.NewSignatureRepository(common.Gorm)
	participantRepo := participantmodel.NewParticipantRepository(common.Gorm, common.Mongo)
	ssoService := util.NewSSOService()

	// Initialize certificate controller with dependencies
	certCtrl := certificate_controller.NewCertificateController(certRepo, signatureRepo, participantRepo)

	certificateGroup := router.Group("certificate")

	certificateGroup.Use(middleware.AuthMiddleware(ssoService))

	certificateGroup.Get("", certCtrl.GetByUser)
	certificateGroup.Get(":certId", certCtrl.GetById)
	certificateGroup.Post("", certCtrl.Create)
	certificateGroup.Put(":id", certCtrl.Update)
	certificateGroup.Delete(":certId", certCtrl.Delete)
	certificateGroup.Post("render/:certId", certCtrl.Render)
	certificateGroup.Get("mail/:certId", certCtrl.DistributeByMail)
	certificateGroup.Post("mail/resend/:participantId", certCtrl.ResendParticipantMail)
	certificateGroup.Get("anchor/:certId", certCtrl.GetAnchorList)
	certificateGroup.Get("generate/status/:certificateId", certCtrl.CheckGenerateStatus)
	certificateGroup.Get("archive/:certId", certCtrl.DownloadArchive)
}
