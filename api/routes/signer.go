package routes

import (
	"github.com/gofiber/fiber/v2"
	signer_controller "github.com/sunthewhat/easy-cert-api/api/controllers/signer"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	signermodel "github.com/sunthewhat/easy-cert-api/api/model/signerModel"
	"github.com/sunthewhat/easy-cert-api/common"
)

func SetupSignerRoutes(router fiber.Router) {
	// Initialize repositories
	signerRepo := signermodel.NewSignerRepository(common.Gorm)
	signatureRepo := signaturemodel.NewSignatureRepository(common.Gorm)
	certificateRepo := certificatemodel.NewCertificateRepository(common.Gorm)

	// Initialize controller with repositories
	signerCtrl := signer_controller.NewSignerController(signerRepo, signatureRepo, certificateRepo)

	signerGroup := router.Group("signer")

	signerGroup.Use(middleware.AuthMiddleware())

	signerGroup.Get("", signerCtrl.GetByUser)
	signerGroup.Post("", signerCtrl.Create)
	signerGroup.Get("status/:certId", signerCtrl.GetStatus)
}
