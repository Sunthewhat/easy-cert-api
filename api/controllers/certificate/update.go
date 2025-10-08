package certificate_controller

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

// extractSignerIdsFromDesign parses the certificate design JSON and extracts all signer IDs
// from objects with ID pattern "SIGNATURE-{UUID}"
func extractSignerIdsFromDesign(designJSON string) ([]string, error) {
	var design map[string]any
	if err := json.Unmarshal([]byte(designJSON), &design); err != nil {
		return nil, err
	}

	objects, ok := design["objects"].([]any)
	if !ok {
		return []string{}, nil
	}

	signerIds := make(map[string]bool)
	for _, obj := range objects {
		objMap, ok := obj.(map[string]any)
		if !ok {
			continue
		}

		id, exists := objMap["id"].(string)
		if exists && strings.HasPrefix(id, "SIGNATURE-") {
			signerId := strings.TrimPrefix(id, "SIGNATURE-")
			signerIds[signerId] = true
		}
	}

	// Convert map to slice for unique signer IDs
	result := make([]string, 0, len(signerIds))
	for signerId := range signerIds {
		result = append(result, signerId)
	}

	return result, nil
}

func Update(c *fiber.Ctx) error {
	// Get certificate ID from URL parameter
	id := c.Params("id")
	if id == "" {
		return response.SendFailed(c, "Certificate ID is required")
	}

	autoSaveQuery := c.Query("autosave")

	isAutoSave := autoSaveQuery == "true"

	shareQuery := c.Query("share")

	isShare := shareQuery == "true"

	// Parse request body
	body := new(payload.UpdateCertificatePayload)
	if err := c.BodyParser(body); err != nil {
		return response.SendError(c, "Failed to parse request body")
	}

	// Validate request body using validator
	if err := util.ValidateStruct(body); err != nil {
		errors := util.GetValidationErrors(err)
		return response.SendFailed(c, errors[0])
	}

	// Validate at least one field is provided for update
	if body.Name == "" && body.Design == "" {
		return response.SendFailed(c, "At least one field (name or design) must be provided")
	}

	// Update certificate
	updatedCert, updateErr := certificatemodel.Update(id, body.Name, body.Design)
	if updateErr != nil {
		if updateErr.Error() == "certificate not found" {
			slog.Warn("Certificate Update attempt with non-existent ID", "cert_id", id)
			return response.SendFailed(c, "Certificate not found")
		}
		slog.Error("Certificate Update controller failed", "error", updateErr, "cert_id", id)
		return response.SendInternalError(c, updateErr)
	}

	// If design was updated, clean up deleted anchors from participants
	if body.Design != "" {
		cleanupErr := participantmodel.CleanupDeletedAnchors(id, updatedCert.Design)
		if cleanupErr != nil {
			slog.Warn("Failed to cleanup deleted anchors from participants", "error", cleanupErr, "cert_id", id)
			// Don't fail the update operation if cleanup fails, just log it
		}
	}

	// If sharing certificate, create signature records for all signature placeholders
	if isShare {
		signerIds, extractErr := extractSignerIdsFromDesign(updatedCert.Design)
		if extractErr != nil {
			slog.Warn("Failed to extract signer IDs from certificate design", "error", extractErr, "cert_id", id)
			// Don't fail the update operation if extraction fails, just log it
		} else if len(signerIds) > 0 {
			// Get user ID from context
			userId, status := middleware.GetUserFromContext(c)
			if !status {
				slog.Warn("Failed to get user ID from context when sharing certificate", "cert_id", id)
				// Don't fail the update operation if user context is missing, just log it
			} else {
				bulkCreateErr := signaturemodel.BulkCreateSignatures(id, signerIds, userId)
				if bulkCreateErr != nil {
					slog.Warn("Failed to create signatures for certificate", "error", bulkCreateErr, "cert_id", id)
					// Don't fail the update operation if signature creation fails, just log it
				} else {
					// Send signature request emails after successful signature creation
					emailErr := signaturemodel.BulkSendSignatureRequests(id, updatedCert.Name, signerIds)
					if emailErr != nil {
						slog.Warn("Failed to send signature request emails", "error", emailErr, "cert_id", id)
						// Don't fail the update operation if email sending fails, just log it
					}
				}
			}
		}
	}

	slog.Info("Certificate Update successful", "cert_id", id, "cert_name", updatedCert.Name)

	if !isAutoSave {
		// Start thumbnail rendering in background - don't block the response
		util.RenderCertificateThumbnailAsync(updatedCert)
	}

	return response.SendSuccess(c, "Certificate updated successfully", updatedCert)
}
