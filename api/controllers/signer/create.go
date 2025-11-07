package signer_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/payload"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func (ctrl *SignerController) Create(c *fiber.Ctx) error {
	body := new(payload.CreateSignerPayload)

	if err := c.BodyParser(body); err != nil {
		return response.SendError(c, "Failed to parse body")
	}

	if err := util.ValidateStruct(body); err != nil {
		errors := util.GetValidationErrors(err)
		return response.SendFailed(c, errors[0])
	}

	userId, status := middleware.GetUserFromContext(c)

	if !status {
		slog.Error("Create Signer failed to get userId from context")
		return response.SendUnauthorized(c, "Invalid token context")
	}

	// Check if email already exists for this creator (not globally)
	existingSigner, err := ctrl.signerRepo.GetByEmail(body.Email, userId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	if existingSigner != nil {
		return response.SendFailed(c, "Signer with this email already existed")
	}

	newSigner, err := ctrl.signerRepo.Create(*body, userId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	return response.SendSuccess(c, "Signer Created", newSigner)
}
