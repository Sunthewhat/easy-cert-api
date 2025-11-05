package routes

import (
	"github.com/gofiber/fiber/v2"
	participant_controller "github.com/sunthewhat/easy-cert-api/api/controllers/participant"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/common"
)

func SetupParticipantRoutes(router fiber.Router) {
	// Initialize repositories
	participantRepo := participantmodel.NewParticipantRepository(common.Gorm, common.Mongo)
	certificateRepo := certificatemodel.NewCertificateRepository(common.Gorm)

	// Initialize controller with repositories
	participantCtrl := participant_controller.NewParticipantController(participantRepo, certificateRepo)

	participantGroup := router.Group("participant")

	participantGroup.Get("validation/:participantId", participantCtrl.GetValidationDataByParticipantId)

	participantGroup.Use(middleware.AuthMiddleware())

	participantGroup.Get(":certId", participantCtrl.GetByCert)
	participantGroup.Post("add/:certId", participantCtrl.Add)
	participantGroup.Put("revoke/:id", participantCtrl.Revoke)
	participantGroup.Put("edit/:id", participantCtrl.EditByID)
	participantGroup.Put("distribute", participantCtrl.UpdateIsDistribute)
	participantGroup.Delete(":id", participantCtrl.Delete)
}
