package certificate_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func (ctrl *CertificateController) GetById(c *fiber.Ctx) error {
	certId := c.Params("certId")

	if certId == "" {
		slog.Warn("Certificate GetById attempt with empty ID")
		return response.SendFailed(c, "Certificate ID is required")
	}

	cert, err := ctrl.certRepo.GetById(certId)

	if err != nil {
		slog.Error("Error getting certificate", "certId", certId, "error", err)
		return response.SendInternalError(c, err)
	}

	if cert == nil {
		slog.Warn("Getting non-existing certificate", "certId", certId)
		return response.SendFailed(c, "Certificate not found")
	}

	slog.Info("Certificate GetById successful", "cert_id", certId, "cert_name", cert.Name)
	return response.SendSuccess(c, "Certificate found", cert)
}
