package certificate_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func GetAll(c *fiber.Ctx) error {
	certificates, err := certificatemodel.GetAll()

	if err != nil {
		slog.Error("Certificate GetAll controller failed", "error", err)
		return response.SendInternalError(c, err)
	}

	slog.Info("Certificate GetAll successful", "count", len(certificates))
	return response.SendSuccess(c, "Certificate fetched", certificates)
}
