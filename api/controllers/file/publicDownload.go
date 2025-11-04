package file

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

// PublicDownloadCertificate serves certificate files publicly for participants
// This endpoint validates the participant ID before serving the file
func PublicDownloadCertificate(c *fiber.Ctx) error {
	participantId := c.Params("participantId")

	if participantId == "" {
		slog.Warn("Public certificate download attempt without participant ID")
		return response.SendFailed(c, "Participant ID is required")
	}

	// Get participant data to validate and get certificate URL
	participant, err := participantmodel.GetParticipantsById(participantId)
	if err != nil {
		slog.Warn("Public certificate download: participant not found", "participant_id", participantId)
		return response.SendError(c, "Certificate not found")
	}

	// Validate participant status
	if participant.IsRevoke {
		slog.Warn("Public certificate download: certificate revoked", "participant_id", participantId)
		return response.SendFailed(c, "This certificate has been revoked")
	}

	if participant.CertificateURL == "" {
		slog.Warn("Public certificate download: no certificate URL", "participant_id", participantId)
		return response.SendError(c, "Certificate not available")
	}

	// Extract object path from certificate URL
	// Handle both direct MinIO URLs and proxy URLs (full or relative paths)
	var objectPath string

	// Check if it's a proxy URL by looking for "/files/download/" pattern
	if strings.Contains(participant.CertificateURL, "/files/download/") {
		// It's a proxy URL, extract object path from it
		// Format: http://localhost:8000/api/v1/files/download/bucket/path/to/file.pdf
		// Or: /api/v1/files/download/bucket/path/to/file.pdf
		// Or: /files/download/bucket/path/to/file.pdf
		parts := strings.Split(participant.CertificateURL, "/files/download/")
		if len(parts) == 2 {
			// Now we have "bucket/path/to/file.pdf"
			remainingPath := parts[1]
			// Skip the bucket name
			bucketPrefix := *common.Config.BucketCertificate + "/"
			if strings.HasPrefix(remainingPath, bucketPrefix) {
				objectPath = strings.TrimPrefix(remainingPath, bucketPrefix)
			} else {
				slog.Error("Invalid proxy URL format - bucket mismatch",
					"participant_id", participantId,
					"certificate_url", participant.CertificateURL)
				return response.SendError(c, "Invalid certificate URL")
			}
		} else {
			slog.Error("Invalid proxy URL format",
				"participant_id", participantId,
				"certificate_url", participant.CertificateURL)
			return response.SendError(c, "Invalid certificate URL")
		}
	} else {
		// It's a direct MinIO URL, extract the object path
		var err error
		objectPath, err = util.ExtractObjectNameFromURL(participant.CertificateURL, *common.Config.BucketCertificate)
		if err != nil {
			slog.Error("Failed to extract object path from certificate URL",
				"error", err,
				"participant_id", participantId,
				"certificate_url", participant.CertificateURL)
			return response.SendInternalError(c, err)
		}
	}

	ctx := context.Background()

	// Download file from MinIO
	object, err := util.DownloadFile(ctx, *common.Config.BucketCertificate, objectPath)
	if err != nil {
		slog.Error("Public certificate download failed",
			"error", err,
			"participant_id", participantId,
			"object_path", objectPath)
		return response.SendError(c, "Certificate file not found")
	}
	defer object.Close()

	// Read the object stats to get content type and size
	objectInfo, err := object.Stat()
	if err != nil {
		slog.Error("Failed to get certificate file stats",
			"error", err,
			"participant_id", participantId,
			"object_path", objectPath)
		return response.SendInternalError(c, err)
	}

	// Determine content type based on file extension
	contentType := "application/octet-stream"
	if strings.HasSuffix(objectPath, ".pdf") {
		contentType = "application/pdf"
	}

	// Extract filename for download
	parts := strings.Split(objectPath, "/")
	filename := parts[len(parts)-1]

	// Set response headers - force download
	c.Set("Content-Type", contentType)
	c.Set("Content-Length", fmt.Sprintf("%d", objectInfo.Size))
	c.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	// Mark as downloaded if this is the first download
	if !participant.IsDownloaded {
		err := participantmodel.MarkAsDownloaded(participantId)
		if err != nil {
			slog.Warn("Failed to mark participant as downloaded",
				"error", err,
				"participant_id", participantId)
			// Don't fail the download if marking fails
		} else {
			slog.Info("Marked participant as downloaded", "participant_id", participantId)
		}
	}

	// Stream the file to the response
	_, err = io.Copy(c.Response().BodyWriter(), object)
	if err != nil {
		slog.Error("Failed to stream certificate file",
			"error", err,
			"participant_id", participantId,
			"object_path", objectPath)
		return response.SendInternalError(c, err)
	}

	slog.Info("Public certificate downloaded successfully",
		"participant_id", participantId,
		"object_path", objectPath,
		"size", objectInfo.Size)

	return nil
}
