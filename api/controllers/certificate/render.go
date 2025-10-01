package certificate_controller

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/internal/renderer"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Render(c *fiber.Ctx) error {
	certId := c.Params("certId")

	if certId == "" {
		slog.Warn("Certificate Render attempt with empty certificate ID")
		return response.SendFailed(c, "Certificate ID is required")
	}

	// Get certificate data
	cert, err := certificatemodel.GetById(certId)
	if err != nil {
		slog.Error("Certificate Render GetById failed", "error", err, "cert_id", certId)
		return response.SendInternalError(c, err)
	}

	if cert == nil {
		slog.Warn("Certificate Render certificate not found", "cert_id", certId)
		return response.SendFailed(c, "Certificate not found")
	}

	if !cert.IsDistributed {
		err := certificatemodel.MarkAsDistributed(certId)
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
	participants, err := participantmodel.GetParticipantsByCertId(certId)
	if err != nil {
		slog.Error("Certificate Render GetParticipantsByCertId failed", "error", err, "cert_id", certId)
		return response.SendInternalError(c, err)
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

	// Convert certificate struct to map for renderer compatibility
	certMap := map[string]any{
		"id":     cert.ID,
		"name":   cert.Name,
		"design": cert.Design,
		// Add other fields as needed
	}

	// Process certificates with embedded renderer
	results, zipFilePath, err := embeddedRenderer.ProcessCertificates(ctx, certMap, participantInterfaces)
	if err != nil {
		slog.Error("Embedded renderer processing failed", "error", err, "cert_id", certId)
		return response.SendError(c, fmt.Sprintf("Renderer processing failed: %v", err))
	}

	// Update certificate archive URL
	if zipFilePath != "" {
		// Use direct path without URL encoding to preserve forward slashes
		archiveURL := fmt.Sprintf("https://%s/%s/%s", *common.Config.MinIoEndpoint, *common.Config.BucketCertificate, zipFilePath)
		certificatemodel.EditArchiveUrl(certId, archiveURL)
		slog.Info("Updated certificate archive URL", "cert_id", certId, "zip_path", zipFilePath, "url", archiveURL)
	}

	// Update participant certificate URLs
	for _, result := range results {
		if result.Status == "success" && result.FilePath != "" {
			// Use direct path without URL encoding to preserve forward slashes
			certificateURL := fmt.Sprintf("https://%s/%s/%s", *common.Config.MinIoEndpoint, *common.Config.BucketCertificate, result.FilePath)
			err := participantmodel.UpdateParticipantCertificateUrlInPostgres(result.ParticipantID, certificateURL)
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
	updatedParticipants, err := participantmodel.GetParticipantsByCertId(certId)
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
