package certificate_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Delete(c *fiber.Ctx) error {
	certId := c.Params("certId")

	if certId == "" {
		slog.Warn("Certificate Delete attempt with empty ID")
		return response.SendFailed(c, "Certificate ID is required")
	}

	cert, err := certificatemodel.GetById(certId)

	if err != nil {
		slog.Error("Error getting certificate", "certId", certId)
		return response.SendInternalError(c, err)
	}

	if cert == nil {
		slog.Warn("Deleting non-existing certificate")
		return response.SendFailed(c, "Certificate not found")
	}

	// Delete participants first
	participants, err := participantmodel.DeleteByCertId(certId)
	if err != nil {
		slog.Error("Deleting participant before certificate", "error", err, "certId", certId)
		return response.SendInternalError(c, err)
	}

	// Delete signatures associated with this certificate
	signatures, err := signaturemodel.DeleteSignaturesByCertificate(certId)
	if err != nil {
		slog.Error("Deleting signatures before certificate", "error", err, "certId", certId)
		return response.SendInternalError(c, err)
	}
	slog.Info("Deleted signatures for certificate", "certId", certId, "count", len(signatures))

	deletedCert, err := certificatemodel.Delete(certId)

	if err != nil {
		slog.Error("Certificate Delete controller failed", "error", err, "cert_id", certId)
		if err.Error() == "certificate not found" {
			return response.SendFailed(c, "Certificate not found")
		}
		return response.SendInternalError(c, err)
	}

	slog.Info("Certificate Delete successful", "cert_id", certId, "cert_name", deletedCert.Name)
	return response.SendSuccess(c, "Certificate Deleted", fiber.Map{
		"certificate":  deletedCert,
		"participants": participants,
		"signatures":   signatures,
	})
}
