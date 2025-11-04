package certificate_controller

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
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

// stringSliceDifference returns elements in slice 'a' that are not in slice 'b'
func stringSliceDifference(a, b []string) []string {
	mb := make(map[string]bool)
	for _, x := range b {
		mb[x] = true
	}
	var diff []string
	for _, x := range a {
		if !mb[x] {
			diff = append(diff, x)
		}
	}
	return diff
}

func (ctrl *CertificateController) Update(c *fiber.Ctx) error {
	// Get certificate ID from URL parameter
	id := c.Params("id")
	if id == "" {
		return response.SendFailed(c, "Certificate ID is required")
	}

	autoSaveQuery := c.Query("autosave")

	isAutoSave := autoSaveQuery == "true"

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
	updatedCert, updateErr := ctrl.certRepo.Update(id, body.Name, body.Design)
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

	// Synchronize signatures when not autosaving (actual save operation)
	if !isAutoSave && body.Design != "" {
		// Extract signer IDs from the updated design
		newSignerIds, extractErr := extractSignerIdsFromDesign(updatedCert.Design)
		slog.Info("Found signers", "signerIds", newSignerIds)
		if extractErr != nil {
			slog.Warn("Certificate Update: Failed to extract signer IDs from design", "error", extractErr, "cert_id", id)
		} else {
			// Get existing signatures for this certificate
			existingSignatures, getErr := signaturemodel.GetSignaturesByCertificate(id)
			if getErr != nil {
				slog.Warn("Certificate Update: Failed to get existing signatures", "error", getErr, "cert_id", id)
			} else {
				// Extract existing signer IDs
				existingSignerIds := make([]string, len(existingSignatures))
				for i, sig := range existingSignatures {
					existingSignerIds[i] = sig.SignerID
				}

				// Find added and removed signer IDs
				addedSignerIds := stringSliceDifference(newSignerIds, existingSignerIds)
				removedSignerIds := stringSliceDifference(existingSignerIds, newSignerIds)

				// Get user ID for creating new signatures
				userId, userStatus := middleware.GetUserFromContext(c)
				if !userStatus {
					slog.Warn("Certificate Update: Failed to get user ID from context", "cert_id", id)
				}

				// Add new signatures for newly added SIGNATURE objects
				if len(addedSignerIds) > 0 && userStatus {
					slog.Info("Certificate Update: Adding new signatures", "cert_id", id, "count", len(addedSignerIds), "signerIds", addedSignerIds)
					createErr := signaturemodel.BulkCreateSignatures(id, addedSignerIds, userId)
					if createErr != nil {
						slog.Warn("Certificate Update: Failed to create new signatures", "error", createErr, "cert_id", id)
					} else {
						// Mark certificate as unsigned since new signatures were added
						markErr := ctrl.certRepo.MarkAsUnsigned(id)
						if markErr != nil {
							slog.Warn("Certificate Update: Failed to mark certificate as unsigned", "error", markErr, "cert_id", id)
						}

						// Send signature request emails for newly added signatures
						emailErr := util.BulkSendSignatureRequests(id, updatedCert.Name, addedSignerIds)
						if emailErr != nil {
							slog.Warn("Certificate Update: Failed to send signature request emails", "error", emailErr, "cert_id", id)
						}
					}
				}

				// Remove signatures for deleted SIGNATURE objects
				if len(removedSignerIds) > 0 {
					slog.Info("Certificate Update: Removing deleted signatures", "cert_id", id, "count", len(removedSignerIds), "signerIds", removedSignerIds)
					for _, signerId := range removedSignerIds {
						deleteErr := signaturemodel.DeleteSignature(id, signerId)
						if deleteErr != nil {
							slog.Warn("Certificate Update: Failed to delete signature", "error", deleteErr, "cert_id", id, "signerId", signerId)
						}
					}

					// After removing signatures, check if all remaining signatures are complete
					allComplete, checkErr := signaturemodel.AreAllSignaturesComplete(id)
					if checkErr != nil {
						slog.Warn("Certificate Update: Failed to check if all signatures complete", "error", checkErr, "cert_id", id)
					} else if allComplete {
						// All remaining signatures are signed, mark certificate as signed and notify owner
						slog.Info("Certificate Update: All remaining signatures are complete after removal", "cert_id", id)

						markErr := ctrl.certRepo.MarkAsSigned(id)
						if markErr != nil {
							slog.Warn("Certificate Update: Failed to mark certificate as signed", "error", markErr, "cert_id", id)
						}

						notifyErr := util.SendAllSignaturesCompleteMail(updatedCert.UserID, updatedCert.Name, updatedCert.ID, "")
						if notifyErr != nil {
							slog.Warn("Certificate Update: Failed to send completion notification", "error", notifyErr, "cert_id", id, "owner", updatedCert.UserID)
						} else {
							slog.Info("Certificate Update: Owner notified of completion", "cert_id", id, "owner", updatedCert.UserID)
						}
					} else {
						// Not all signatures complete, ensure certificate is marked as unsigned
						markErr := ctrl.certRepo.MarkAsUnsigned(id)
						if markErr != nil {
							slog.Warn("Certificate Update: Failed to mark certificate as unsigned", "error", markErr, "cert_id", id)
						}
					}
				}

				if len(addedSignerIds) == 0 && len(removedSignerIds) == 0 {
					slog.Info("Certificate Update: No signature changes detected", "cert_id", id)
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
