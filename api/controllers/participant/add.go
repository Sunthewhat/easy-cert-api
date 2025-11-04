package participant_controller

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Add(c *fiber.Ctx) error {
	certId := c.Params("certId")

	if certId == "" {
		slog.Warn("Participant Add attempt with empty certificate ID")
		return response.SendFailed(c, "Certificate ID is required")
	}

	// First verify that the certificate exists
	certRepo := certificatemodel.NewCertificateRepository(common.Gorm)
	cert, err := certRepo.GetById(certId)
	if err != nil {
		slog.Error("Participant Add certificate verification failed", "error", err, "cert_id", certId)
		return response.SendInternalError(c, err)
	}
	if cert == nil {
		slog.Warn("Participant Add attempt with non-existent certificate", "cert_id", certId)
		return response.SendFailed(c, "Certificate not found")
	}

	// Parse request body
	body := new(payload.AddParticipantPayload)
	if err := c.BodyParser(body); err != nil {
		slog.Error("Participant Add body parsing failed", "error", err, "cert_id", certId)
		return response.SendError(c, "Failed to parse body")
	}

	// Validate request structure
	if err := util.ValidateStruct(body); err != nil {
		errors := util.GetValidationErrors(err)
		slog.Warn("Participant Add validation failed", "error", errors[0], "cert_id", certId)
		return response.SendFailed(c, errors[0])
	}

	// Note: Field validation against certificate design anchors is now handled in the model layer

	// Check if collection already exists and has documents
	count, countErr := participantmodel.GetParticipantCollectionCount(certId)
	if countErr != nil {
		slog.Error("Participant Add collection count failed", "error", countErr, "cert_id", certId)
		return response.SendInternalError(c, countErr)
	}

	// Log collection status
	if count > 0 {
		slog.Info("Participant Add found existing collection", "cert_id", certId, "existing_count", count, "new_count", len(body.Participants))
	} else {
		slog.Info("Participant Add creating new collection", "cert_id", certId, "participant_count", len(body.Participants))
	}

	// Add participants using model function
	result, addErr := participantmodel.AddParticipants(certId, body.Participants)
	if addErr != nil {
		slog.Error("Participant Add failed", "error", addErr, "cert_id", certId)
		return response.SendInternalError(c, addErr)
	}

	collectionName := "participant-" + certId
	totalParticipants := count + int64(len(result.CreatedIDs))

	slog.Info("Participant Add controller successful",
		"cert_id", certId,
		"requested_count", len(body.Participants),
		"mongo_created", len(result.MongoResult.InsertedIDs),
		"postgres_created", len(result.PostgresRecords),
		"fully_created", len(result.CreatedIDs),
		"total_participants", totalParticipants)

	// Build response with dual-database information
	responseData := fiber.Map{
		"certificate_id":       certId,
		"collection_name":      collectionName,
		"requested_count":      len(body.Participants),
		"successfully_created": len(result.CreatedIDs),
		"total_participants":   totalParticipants,
		"created_ids":          result.CreatedIDs,
		"databases": fiber.Map{
			"mongodb": fiber.Map{
				"collection":     collectionName,
				"inserted_count": len(result.MongoResult.InsertedIDs),
			},
			"postgresql": fiber.Map{
				"created_count": len(result.PostgresRecords),
				"failed_count":  len(result.FailedPostgresIDs),
			},
		},
	}

	// Add warning info if there were PostgreSQL failures
	if len(result.FailedPostgresIDs) > 0 {
		responseData["warnings"] = []string{
			fmt.Sprintf("%d participants were created in MongoDB but failed in PostgreSQL indexing", len(result.FailedPostgresIDs)),
		}
		responseData["failed_postgres_ids"] = result.FailedPostgresIDs
	}

	message := "Participants added successfully"
	if len(result.FailedPostgresIDs) > 0 {
		message = "Participants added with some PostgreSQL indexing failures"
	}

	return response.SendSuccess(c, message, responseData)
}
