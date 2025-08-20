package routes

import (
	"github.com/gofiber/fiber/v2"
	participant_controller "github.com/sunthewhat/easy-cert-api/api/controllers/participant"
)

func SetupParticipantRoutes(router fiber.Router) {
	participantGroup := router.Group("participant")

	participantGroup.Post("add/:certId", participant_controller.Add)
	participantGroup.Put(":id/revoke", participant_controller.Revoke)
}
