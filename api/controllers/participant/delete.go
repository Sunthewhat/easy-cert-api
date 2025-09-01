package participant_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Delete(c *fiber.Ctx) error {
	// Get participant ID from URL parameter
	id := c.Params("id")
	if id == "" {
		return response.SendFailed(c, "Participant ID is required")
	}

	// Delete the participant from both databases
	deletedParticipant, err := participantmodel.DeleteParticipantByID(id)
	if err != nil {
		slog.Error("Participant Delete controller", "error", err, "participant_id", id)
		return response.SendFailed(c, "Failed to delete participant: "+err.Error())
	}

	slog.Info("Participant Delete controller success", "participant_id", id, "certificate_id", deletedParticipant.CertificateID)
	return response.SendSuccess(c, "Participant deleted successfully", deletedParticipant)
}
