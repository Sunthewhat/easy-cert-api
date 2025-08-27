package participant_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/common/util"
)

type EditParticipantPayload struct {
	Data map[string]any `json:"data" validate:"required"`
}

func EditByID(c *fiber.Ctx) error {
	participantId := c.Params("id")
	
	var payload EditParticipantPayload
	if err := c.BodyParser(&payload); err != nil {
		slog.Warn("EditParticipant: Failed to parse request body", "error", err, "participant_id", participantId)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := util.ValidateStruct(payload); err != nil {
		slog.Warn("EditParticipant: Validation failed", "error", err, "participant_id", participantId)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Validation failed",
			"details": util.GetValidationErrors(err),
		})
	}

	updatedParticipant, err := participantmodel.EditParticipantByID(participantId, payload.Data)
	if err != nil {
		slog.Error("EditParticipant: Failed to update participant", "error", err, "participant_id", participantId)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update participant",
			"details": err.Error(),
		})
	}

	slog.Info("EditParticipant: Successfully updated participant", "participant_id", participantId)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Participant updated successfully",
		"data": updatedParticipant,
	})
}