package routes

import (
	"github.com/gofiber/fiber/v2"
	participant_controller "github.com/sunthewhat/easy-cert-api/api/controllers/participant"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
)

func SetupParticipantRoutes(router fiber.Router) {
	// Initialize repositories
	participantRepo := participantmodel.NewParticipantRepository(common.Gorm, common.Mongo)
	certificateRepo := certificatemodel.NewCertificateRepository(common.Gorm)
	ssoService := util.NewSSOService()

	// Initialize controller with repositories
	participantCtrl := participant_controller.NewParticipantController(participantRepo, certificateRepo)

	participantGroup := router.Group("participant")

	participantGroup.Get("validation/:participantId", participantCtrl.GetValidationDataByParticipantId)

	participantGroup.Use(middleware.AuthMiddleware(ssoService))

	participantGroup.Get(":certId", participantCtrl.GetByCert)
	participantGroup.Post("add/:certId", participantCtrl.Add)
	participantGroup.Put("revoke/:id", participantCtrl.Revoke)
	participantGroup.Put("edit/:id", participantCtrl.EditByID)
	participantGroup.Delete(":id", participantCtrl.Delete)
}
