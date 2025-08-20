package certificate_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Create(c *fiber.Ctx) error {
	body := new(payload.CreateCertificatePayload)

	if err := c.BodyParser(body); err != nil {
		return response.SendError(c, "Failed to parse body")
	}

	if err := util.ValidateStruct(body); err != nil {
		errors := util.GetValidationErrors(err)
		return response.SendFailed(c, errors[0])
	}

	userId, status := middleware.GetUserFromContext(c)

	if !status {
		slog.Error("Certificate Create GetUserId failed")
		return response.SendError(c, "Failed to read user")
	}

	newCert, err := certificatemodel.Create(*body, userId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	return response.SendSuccess(c, "Certificate Created", newCert)
}
