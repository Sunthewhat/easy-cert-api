package certificate_controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/internal/renderer"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func (ctrl *CertificateController) Render(c *fiber.Ctx) error {
	certId := c.Params("certId")

	isRenewAll := c.Query("renew") // "true", "false", ""

	if certId == "" {
		slog.Warn("Certificate Render attempt with empty certificate ID")
		return response.SendFailed(c, "Certificate ID is required")
	}

	// Get certificate data
	cert, err := ctrl.certRepo.GetById(certId)
	if err != nil {
		slog.Error("Certificate Render GetById failed", "error", err, "cert_id", certId)
		return response.SendInternalError(c, err)
	}

	if cert == nil {
		slog.Warn("Certificate Render certificate not found", "cert_id", certId)
		return response.SendFailed(c, "Certificate not found")
	}

	if !cert.IsDistributed {
		err := ctrl.certRepo.MarkAsDistributed(certId)
		if err != nil {
			return response.SendInternalError(c, err)
		}
	}

	userId, success := middleware.GetUserFromContext(c)

	if !success {
		slog.Error("Certificate Render UserId not found in context")
		return response.SendUnauthorized(c, "Unknown user request")
	}

	if userId != cert.UserID {
		slog.Warn("Wrong Owner Request Render", "user", userId, "certificate-owner", cert.UserID)
		return response.SendUnauthorized(c, "User did not own this certificate")
	}

	// Get participants data
	allParticipants, err := ctrl.participantRepo.GetParticipantsByCertId(certId)
	if err != nil {
		slog.Error("Certificate Render GetParticipantsByCertId failed", "error", err, "cert_id", certId)
		return response.SendInternalError(c, err)
	}

	// Get all signatures for this certificate
	signatures, sigErr := ctrl.signatureRepo.GetSignaturesByCertificate(certId)
	if sigErr != nil {
		slog.Error("Certificate Render GetSignaturesByCertificate failed", "error", sigErr, "cert_id", certId)
		return response.SendInternalError(c, sigErr)
	}

	// Decrypt signature images and create a map of signerId -> base64 image
	decryptedSignatures := make(map[string]string)
	for _, sig := range signatures {
		if sig.IsSigned && sig.Signature != "" {
			decryptedImage, decryptErr := util.DecryptData(sig.Signature, *common.Config.EncryptionKey)
			if decryptErr != nil {
				slog.Warn("Certificate Render: Failed to decrypt signature",
					"error", decryptErr,
					"cert_id", certId,
					"signer_id", sig.SignerID)
				continue
			}
			// Convert to base64 for rendering
			decryptedSignatures[sig.SignerID] = base64.StdEncoding.EncodeToString(decryptedImage)
		}
	}

	slog.Info("Certificate Render: Decrypted signatures",
		"cert_id", certId,
		"total_signatures", len(signatures),
		"decrypted_count", len(decryptedSignatures))

	// Filter participants based on isRenewAll parameter
	var participants []*participantmodel.CombinedParticipant
	if isRenewAll == "true" {
		// Renew all participants
		participants = allParticipants
		slog.Info("Certificate Render: Renewing all participants", "cert_id", certId, "count", len(participants))
	} else {
		// Only renew participants that haven't been distributed
		for _, p := range allParticipants {
			if p.EmailStatus != "success" {
				participants = append(participants, p)
			}
		}
		slog.Info("Certificate Render: Renewing only non-distributed participants",
			"cert_id", certId,
			"total_count", len(allParticipants),
			"to_renew_count", len(participants))
	}

	// Reset participant statuses (email_status to "pending" and is_downloaded to false)
	if len(participants) > 0 {
		participantIds := make([]string, len(participants))
		for i, p := range participants {
			participantIds[i] = p.ID
		}

		err = ctrl.participantRepo.ResetParticipantStatuses(participantIds)
		if err != nil {
			slog.Warn("Certificate Render: Failed to reset participant statuses",
				"error", err,
				"cert_id", certId,
				"participant_count", len(participantIds))
			// Don't fail the operation, just log the warning
		} else {
			slog.Info("Certificate Render: Reset participant statuses successfully",
				"cert_id", certId,
				"participant_count", len(participantIds))
		}
	}

	// Delete old zip archive file
	if cert.ArchiveURL != "" {
		slog.Info("Certificate Render: Deleting old zip archive",
			"cert_id", certId,
			"archive_url", cert.ArchiveURL)

		ctx := context.Background()
		err := util.DeleteFileByURL(ctx, *common.Config.BucketCertificate, cert.ArchiveURL)
		if err != nil {
			slog.Warn("Certificate Render: Failed to delete old zip archive",
				"error", err,
				"cert_id", certId,
				"archive_url", cert.ArchiveURL)
		} else {
			slog.Info("Certificate Render: Successfully deleted old zip archive",
				"cert_id", certId)
		}
	}

	// Delete old certificate files for participants that will be regenerated
	if len(participants) > 0 {
		slog.Info("Certificate Render: Deleting old certificate files",
			"cert_id", certId,
			"participant_count", len(participants))

		ctx := context.Background()
		deletedCount := 0
		failedCount := 0

		for _, p := range participants {
			if p.CertificateURL != "" {
				err := util.DeleteFileByURL(ctx, *common.Config.BucketCertificate, p.CertificateURL)
				if err != nil {
					slog.Warn("Certificate Render: Failed to delete old certificate file",
						"error", err,
						"participant_id", p.ID,
						"certificate_url", p.CertificateURL)
					failedCount++
				} else {
					slog.Debug("Certificate Render: Deleted old certificate file",
						"participant_id", p.ID,
						"certificate_url", p.CertificateURL)
					deletedCount++
				}
			}
		}

		slog.Info("Certificate Render: Old certificate cleanup completed",
			"cert_id", certId,
			"deleted_count", deletedCount,
			"failed_count", failedCount)
	}

	slog.Info("Certificate Render starting embedded renderer",
		"cert_id", certId,
		"participant_count", len(participants),
		"estimated_time", "This may take several minutes for large batches")

	// Initialize embedded renderer
	embeddedRenderer, err := renderer.NewEmbeddedRenderer()
	if err != nil {
		slog.Error("Failed to initialize embedded renderer", "error", err, "cert_id", certId)
		return response.SendError(c, "Failed to initialize renderer")
	}
	defer embeddedRenderer.Close()

	// Create context with timeout (reduced from 5min to 2min for embedded renderer)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Convert participants to interface{} slice
	participantInterfaces := make([]any, len(participants))
	for i, p := range participants {
		participantInterfaces[i] = p
	}

	if *common.Config.Environment {
		cert.Design = strings.ReplaceAll(
			cert.Design,
			"http://easycert.sit.kmutt.ac.th",
			"http://backend:8000",
		)
	}

	// Convert certificate struct to map for renderer compatibility
	certMap := map[string]any{
		"id":     cert.ID,
		"name":   cert.Name,
		"design": cert.Design,
		// Add other fields as needed
	}

	// Process certificates with embedded renderer, passing decrypted signatures
	results, zipFilePath, err := embeddedRenderer.ProcessCertificates(ctx, certMap, participantInterfaces, decryptedSignatures)
	if err != nil {
		slog.Error("Embedded renderer processing failed", "error", err, "cert_id", certId)
		return response.SendError(c, fmt.Sprintf("Renderer processing failed: %v", err))
	}

	// Update certificate archive URL with proxy URL
	if zipFilePath != "" {
		// Use backend proxy URL instead of direct MinIO URL for security
		archiveURL := util.GenerateProxyURL(*common.Config.BucketCertificate, zipFilePath)
		ctrl.certRepo.EditArchiveUrl(certId, archiveURL)
		slog.Info("Updated certificate archive URL", "cert_id", certId, "zip_path", zipFilePath, "url", archiveURL)
	}

	// Update participant certificate URLs with proxy URLs
	for _, result := range results {
		if result.Status == "success" && result.FilePath != "" {
			// Use backend proxy URL instead of direct MinIO URL for security
			certificateURL := util.GenerateProxyURL(*common.Config.BucketCertificate, result.FilePath)
			err := ctrl.participantRepo.UpdateParticipantCertificateUrl(result.ParticipantID, certificateURL)
			if err != nil {
				slog.Warn("Certificate Render failed to update participant certificate URL",
					"error", err,
					"participant_id", result.ParticipantID,
					"file_path", result.FilePath)
			} else {
				slog.Info("Certificate Render updated participant certificate URL",
					"participant_id", result.ParticipantID,
					"file_path", result.FilePath,
					"url", certificateURL)
			}
		}
	}

	// Get updated participants data
	updatedParticipants, err := ctrl.participantRepo.GetParticipantsByCertId(certId)
	if err != nil {
		slog.Error("Certificate Render failed to get updated participants", "error", err, "cert_id", certId)
		// Fallback to results if getting updated participants fails
		return response.SendSuccess(c, "Certificate rendered successfully", map[string]any{
			"results":     results,
			"zipFilePath": zipFilePath,
		})
	}

	slog.Info("Certificate Render completed successfully",
		"cert_id", certId,
		"successful_renders", len(results),
		"zip_file", zipFilePath)

	// Return updated participants with zipFilePath
	return response.SendSuccess(c, "Certificate rendered successfully", map[string]any{
		"participants": updatedParticipants,
		"zipFilePath":  zipFilePath,
	})
}
