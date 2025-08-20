package certificate_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func GetByUser(c *fiber.Ctx) error {
	userId, success := middleware.GetUserFromContext(c)

	if !success {
		slog.Error("Certificate GeyByUser UserToken not found")
		return response.SendUnauthorized(c, "User token not found")
	}

	certificates, err := certificatemodel.GetByUser(userId)

	if err != nil {
		slog.Error("Certificate GetAll controller failed", "error", err)
		return response.SendInternalError(c, err)
	}

	slog.Info("Certificate GetAll successful", "count", len(certificates))
	return response.SendSuccess(c, "Certificate fetched", certificates)
}
