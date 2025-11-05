package certificate_controller

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

// DownloadArchive serves the certificate archive (zip file) and marks all participants as downloaded
func (ctrl *CertificateController) DownloadArchive(c *fiber.Ctx) error {
	certId := c.Params("certId")

	if certId == "" {
		slog.Warn("Certificate archive download attempt without certificate ID")
		return response.SendFailed(c, "Certificate ID is required")
	}

	// Get certificate to retrieve archive URL
	cert, err := ctrl.certRepo.GetById(certId)
	if err != nil {
		slog.Error("Failed to get certificate for archive download", "error", err, "cert_id", certId)
		return response.SendInternalError(c, err)
	}

	if cert == nil {
		slog.Warn("Certificate archive download: certificate not found", "cert_id", certId)
		return response.SendFailed(c, "Certificate not found")
	}

	if cert.ArchiveURL == "" {
		slog.Warn("Certificate archive download: no archive URL", "cert_id", certId)
		return response.SendFailed(c, "Certificate archive not available")
	}

	// Extract object path from archive URL
	var objectPath string

	// Check if it's a proxy URL by looking for "/files/download/" pattern
	if strings.Contains(cert.ArchiveURL, "/files/download/") {
		// It's a proxy URL, extract object path from it
		// Format: http://localhost:8000/api/v1/files/download/bucket/path/to/file.zip
		parts := strings.Split(cert.ArchiveURL, "/files/download/")
		if len(parts) == 2 {
			// Now we have "bucket/path/to/file.zip"
			remainingPath := parts[1]
			// Skip the bucket name
			bucketPrefix := *common.Config.BucketCertificate + "/"
			if strings.HasPrefix(remainingPath, bucketPrefix) {
				objectPath = strings.TrimPrefix(remainingPath, bucketPrefix)
			} else {
				slog.Error("Invalid proxy URL format - bucket mismatch",
					"cert_id", certId,
					"archive_url", cert.ArchiveURL)
				return response.SendError(c, "Invalid archive URL")
			}
		} else {
			slog.Error("Invalid proxy URL format",
				"cert_id", certId,
				"archive_url", cert.ArchiveURL)
			return response.SendError(c, "Invalid archive URL")
		}
	} else {
		// It's a direct MinIO URL, extract the object path
		var extractErr error
		objectPath, extractErr = util.ExtractObjectNameFromURL(cert.ArchiveURL, *common.Config.BucketCertificate)
		if extractErr != nil {
			slog.Error("Failed to extract object path from archive URL",
				"error", extractErr,
				"cert_id", certId,
				"archive_url", cert.ArchiveURL)
			return response.SendInternalError(c, extractErr)
		}
	}

	ctx := context.Background()

	// Download file from MinIO
	object, err := util.DownloadFile(ctx, *common.Config.BucketCertificate, objectPath)
	if err != nil {
		slog.Error("Certificate archive download failed",
			"error", err,
			"cert_id", certId,
			"object_path", objectPath)
		return response.SendError(c, "Archive file not found")
	}
	defer object.Close()

	// Read the object stats to get content type and size
	objectInfo, err := object.Stat()
	if err != nil {
		slog.Error("Failed to get archive file stats",
			"error", err,
			"cert_id", certId,
			"object_path", objectPath)
		return response.SendInternalError(c, err)
	}

	// Extract filename for download
	pathParts := strings.Split(objectPath, "/")
	filename := pathParts[len(pathParts)-1]

	// Set response headers - force download
	c.Set("Content-Type", "application/zip")
	c.Set("Content-Length", fmt.Sprintf("%d", objectInfo.Size))
	c.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	// Mark all participants of this certificate as downloaded
	// This runs asynchronously to not block the download
	go func() {
		participants, getErr := ctrl.participantRepo.GetParticipantsByCertId(certId)
		if getErr != nil {
			slog.Error("Failed to get participants for marking as downloaded",
				"error", getErr,
				"cert_id", certId)
			return
		}

		successCount := 0
		failCount := 0

		for _, participant := range participants {
			if !participant.IsDownloaded {
				markErr := ctrl.participantRepo.MarkAsDownloaded(participant.ID)
				if markErr != nil {
					slog.Error("Failed to mark participant as downloaded",
						"error", markErr,
						"cert_id", certId,
						"participant_id", participant.ID)
					failCount++
				} else {
					successCount++
				}
			}
		}

		slog.Info("Certificate archive download: marked participants as downloaded",
			"cert_id", certId,
			"total_participants", len(participants),
			"marked_as_downloaded", successCount,
			"failed", failCount)
	}()

	// Stream the file to the response
	_, err = io.Copy(c.Response().BodyWriter(), object)
	if err != nil {
		slog.Error("Failed to stream archive file",
			"error", err,
			"cert_id", certId,
			"object_path", objectPath)
		return response.SendInternalError(c, err)
	}

	slog.Info("Certificate archive downloaded successfully",
		"cert_id", certId,
		"object_path", objectPath,
		"size", objectInfo.Size)

	return nil
}
