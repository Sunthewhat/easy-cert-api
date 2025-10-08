package participant_controller

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/gofiber/fiber/v2"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

type EditParticipantPayload struct {
	Data map[string]any `json:"data" validate:"required"`
}

func EditByID(c *fiber.Ctx) error {
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

	updatedParticipant, err := participantmodel.EditParticipantByID(participantId, payload.Data)
	if err != nil {
		slog.Error("EditParticipant: Failed to update participant", "error", err, "participant_id", participantId)
		return response.SendInternalError(c, err)
	}

	slog.Info("EditParticipant: Successfully updated participant", "participant_id", participantId)

	return response.SendSuccess(c, "Participant updated successfully", updatedParticipant)
}

func UpdateIsDistribute(c *fiber.Ctx) error {
	var payload payload.UpdateParticipantIsDistributed
	if err := c.BodyParser(&payload); err != nil {
		slog.Warn("Update Is Distributed failed to parse body", "error", err)
		return response.SendFailed(c, "Invalid request body")
	}

	if err := util.ValidateStruct(payload); err != nil {
		slog.Warn("Update Is Distributed failed to validate", "error", err)
		return response.SendFailed(c, fmt.Sprintf("Invalid Data type %s", util.GetValidationErrors(err)[0]))
	}

	// Get participant IDs directly from the payload
	participantIds := payload.Ids

	// Process updates in parallel using goroutines
	var wg sync.WaitGroup
	var mu sync.Mutex
	var successResults []string
	var failedResults []map[string]string

	// Use buffered channel to limit concurrent goroutines
	maxConcurrency := 10
	semaphore := make(chan struct{}, maxConcurrency)

	for _, participantId := range participantIds {
		if participantId == "" {
			continue // Skip empty IDs
		}

		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			err := participantmodel.MarkParticipantAsDistributed(id)

			participantmodel.UpdateEmailStatus(id, "downloaded")

			// Thread-safe result storage
			mu.Lock()
			if err != nil {
				failedResults = append(failedResults, map[string]string{
					"participant_id": id,
					"error":          err.Error(),
				})
				slog.Error("Failed to mark participant as distributed", "error", err, "participant_id", id)
			} else {
				successResults = append(successResults, id)
				slog.Info("Successfully marked participant as distributed", "participant_id", id)
			}
			mu.Unlock()
		}(participantId)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Prepare response
	responseData := map[string]any{
		"total_participants": len(participantIds),
		"success_count":      len(successResults),
		"failed_count":       len(failedResults),
		"success_results":    successResults,
		"failed_results":     failedResults,
	}

	slog.Info("Update Is Distributed completed",
		"total", len(participantIds),
		"success", len(successResults),
		"failed", len(failedResults))

	return response.SendSuccess(c, "Participants distribution status updated", responseData)
}
