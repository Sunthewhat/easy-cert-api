package certificate_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Delete(c *fiber.Ctx) error {
	certId := c.Params("certId")

	if certId == "" {
		slog.Warn("Certificate Delete attempt with empty ID")
		return response.SendFailed(c, "Certificate ID is required")
	}

	cert, err := certificatemodel.Delete(certId)

	if err != nil {
		slog.Error("Certificate Delete controller failed", "error", err, "cert_id", certId)
		if err.Error() == "certificate not found" {
			return response.SendFailed(c, "Certificate not found")
		}
		return response.SendInternalError(c, err)
	}

	slog.Info("Certificate Delete successful", "cert_id", certId, "cert_name", cert.Name)
	return response.SendSuccess(c, "Certificate Deleted", cert)
}
