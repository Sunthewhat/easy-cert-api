package routes

import (
	"github.com/gofiber/fiber/v2"
	signature_controller "github.com/sunthewhat/easy-cert-api/api/controllers/signature"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	signermodel "github.com/sunthewhat/easy-cert-api/api/model/signerModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
)

func SetupSignatureRoutes(router fiber.Router) {
	// Initialize repositories
	signatureRepo := signaturemodel.NewSignatureRepository(common.Gorm)
	certificateRepo := certificatemodel.NewCertificateRepository(common.Gorm)
	signerRepo := signermodel.NewSignerRepository(common.Gorm)
	participantRepo := participantmodel.NewParticipantRepository(common.Gorm, common.Mongo)
	ssoService := util.NewSSOService()

	// Initialize controller with repositories
	signatureCtrl := signature_controller.NewSignatureController(signatureRepo, certificateRepo, signerRepo, participantRepo)

	signatureGroup := router.Group("signature")

	signatureGroup.Use(middleware.AuthMiddleware(ssoService))

	signatureGroup.Post("", signatureCtrl.Create)
	signatureGroup.Get("resign/:signatureId", signatureCtrl.RequestResign)
	signatureGroup.Get("signer/:certificateId", signatureCtrl.GetSignerData)
	signatureGroup.Get(":id", signatureCtrl.GetById)
	signatureGroup.Put("sign/:id", signatureCtrl.Sign)
	signatureGroup.Get(":certificateId/:signerId", signatureCtrl.GetSignatureImage)
}
