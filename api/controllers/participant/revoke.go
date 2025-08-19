package participant_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Revoke(c *fiber.Ctx) error {
	// Get participant ID from URL parameter
	id := c.Params("id")
	if id == "" {
		return response.SendFailed(c, "Participant ID is required")
	}

	// Revoke the participant
	revokedParticipant, err := participantmodel.Revoke(id)
	if err != nil {
		slog.Error("Participant Revoke controller", "error", err)
		return response.SendFailed(c, "Participant not found or already revoked")
	}

	return response.SendSuccess(c, "Participant revoked successfully", revokedParticipant)
}