package certificate_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Update(c *fiber.Ctx) error {
	// Get certificate ID from URL parameter
	id := c.Params("id")
	if id == "" {
		return response.SendFailed(c, "Certificate ID is required")
	}

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

	slog.Info("Certificate Update successful", "cert_id", id, "cert_name", updatedCert.Name)

	thumbnailErr := util.RenderCertificateThumbnail(updatedCert)

	if thumbnailErr != nil {
		return response.SendInternalError(c, thumbnailErr)
	}

	return response.SendSuccess(c, "Certificate updated successfully", updatedCert)
}
