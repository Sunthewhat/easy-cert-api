package routes

import (
	"github.com/gofiber/fiber/v2"
	participant_controller "github.com/sunthewhat/easy-cert-api/api/controllers/participant"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
)

func SetupParticipantRoutes(router fiber.Router) {
	participantGroup := router.Group("participant")

	participantGroup.Use(middleware.AuthMiddleware())

	participantGroup.Get(":certId", participant_controller.GetByCert)
	participantGroup.Post("add/:certId", participant_controller.Add)
	participantGroup.Put("revoke/:id", participant_controller.Revoke)
	participantGroup.Put("edit/:id", participant_controller.EditByID)
}
