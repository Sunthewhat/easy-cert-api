package certificate_controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/type/payload"
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

	// Prepare request body for renderer
	requestBody := map[string]any{
		"certificate":  cert,
		"participants": participants,
	}

	// Marshal request body to JSON
	jsonData, marshalErr := json.Marshal(requestBody)
	if marshalErr != nil {
		slog.Error("Certificate Render JSON marshal failed", "error", marshalErr, "cert_id", certId)
		return response.SendInternalError(c, marshalErr)
	}

	// Construct renderer URL
	rendererURL := fmt.Sprintf("%s/api/render", *common.Config.RendererUrl)

	slog.Info("Certificate Render sending request to renderer",
		"cert_id", certId,
		"renderer_url", rendererURL,
		"participant_count", len(participants),
		"estimated_time", "This may take several minutes for large batches")

	// Create HTTP client with extended timeout for rendering
	client := &http.Client{
		Timeout: 300 * time.Second, // 5 minutes timeout for rendering
	}

	// Create POST request
	req, reqErr := http.NewRequest("POST", rendererURL, bytes.NewBuffer(jsonData))
	if reqErr != nil {
		slog.Error("Certificate Render request creation failed", "error", reqErr, "cert_id", certId)
		return response.SendInternalError(c, reqErr)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, respErr := client.Do(req)
	if respErr != nil {
		slog.Error("Certificate Render HTTP request failed", "error", respErr, "cert_id", certId, "url", rendererURL)
		return response.SendError(c, "Failed to communicate with renderer service")
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		slog.Error("Certificate Render response read failed", "error", readErr, "cert_id", certId)
		return response.SendError(c, "Failed to read renderer response")
	}

	// Check if request was successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Info("Certificate Render successful",
			"cert_id", certId,
			"status_code", resp.StatusCode,
			"response_size", len(responseBody))

		// Try to parse response as JSON
		var rendererResponse payload.RenderCertificatePayload
		if parseErr := json.Unmarshal(responseBody, &rendererResponse); parseErr == nil {
			certificatemodel.EditArchiveUrl(certId, fmt.Sprintf("https://%s/%s/%s", *common.Config.MinIoEndpoint, *common.Config.BucketCertificate, rendererResponse.ZipFilePath))
			// Update participant certificate URLs
			for _, result := range rendererResponse.Results {
				if result.Status == "success" && result.FilePath != "" {
					err := participantmodel.UpdateParticipantCertificateUrlInPostgres(result.ParticipantId, fmt.Sprintf("https://%s/%s/%s", *common.Config.MinIoEndpoint, *common.Config.BucketCertificate, result.FilePath))
					if err != nil {
						slog.Warn("Certificate Render failed to update participant certificate URL",
							"error", err,
							"participant_id", result.ParticipantId,
							"file_path", result.FilePath)
					} else {
						slog.Info("Certificate Render updated participant certificate URL",
							"participant_id", result.ParticipantId,
							"file_path", result.FilePath)
					}
				}
			}

			// Get updated participants data
			updatedParticipants, err := participantmodel.GetParticipantsByCertId(certId)
			if err != nil {
				slog.Error("Certificate Render failed to get updated participants", "error", err, "cert_id", certId)
				// Fallback to original response if getting updated participants fails
				return response.SendSuccess(c, "Certificate rendered successfully", rendererResponse)
			}

			// Return updated participants with zipFilePath
			return response.SendSuccess(c, "Certificate rendered successfully", map[string]any{
				"participants": updatedParticipants,
				"zipFilePath":  rendererResponse.ZipFilePath,
			})
		} else {
			// If not JSON, return raw response
			return response.SendSuccess(c, "Certificate rendered successfully", map[string]any{
				"response": string(responseBody),
			})
		}
	} else {
		slog.Error("Certificate Render renderer error",
			"cert_id", certId,
			"status_code", resp.StatusCode,
			"response", string(responseBody))

		return response.SendError(c, fmt.Sprintf("Renderer service error: %s", string(responseBody)))
	}
}
