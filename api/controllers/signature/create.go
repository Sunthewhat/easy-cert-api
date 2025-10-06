package signature_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Create(c *fiber.Ctx) error {
	body := new(payload.CreateSignaturePayload)

	if err := c.BodyParser(body); err != nil {
		return response.SendError(c, "Failed to parse body")
	}

	if err := util.ValidateStruct(body); err != nil {
		errors := util.GetValidationErrors(err)
		return response.SendFailed(c, errors[0])
	}

	userId, status := middleware.GetUserFromContext(c)

	if !status {
		slog.Error("Signature Create GetUserId failed")
		return response.SendError(c, "Failed to read user")
	}

	newSignature, err := signaturemodel.Create(*body, userId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	return response.SendSuccess(c, "Signature Created", newSignature)
}
