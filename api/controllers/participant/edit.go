package participant_controller

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

type EditParticipantPayload struct {
	Data map[string]any `json:"data" validate:"required"`
}

func (ctrl *ParticipantController) EditByID(c *fiber.Ctx) error {
	participantId := c.Params("id")

	var payload EditParticipantPayload
	if err := c.BodyParser(&payload); err != nil {
		slog.Warn("EditParticipant: Failed to parse request body", "error", err, "participant_id", participantId)
		return response.SendFailed(c, "Invalid request body")
	}

	if err := util.ValidateStruct(payload); err != nil {
		slog.Warn("EditParticipant: Validation failed", "error", err, "participant_id", participantId)
		return response.SendFailed(c, fmt.Sprintf("Invalid Data type %s", util.GetValidationErrors(err)[0]))
	}

	updatedParticipant, err := ctrl.participantRepo.EditParticipantByID(participantId, payload.Data)
	if err != nil {
		slog.Error("EditParticipant: Failed to update participant", "error", err, "participant_id", participantId)
		return response.SendInternalError(c, err)
	}

	slog.Info("EditParticipant: Successfully updated participant", "participant_id", participantId)

	return response.SendSuccess(c, "Participant updated successfully", updatedParticipant)
}

