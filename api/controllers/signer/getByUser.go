package signer_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func (ctrl *SignerController) GetByUser(c *fiber.Ctx) error {
	userId, success := middleware.GetUserFromContext(c)

	if !success {
		slog.Error("Signer Get By User User not found from context")
		return response.SendUnauthorized(c, "User context failed")
	}

	signers, err := ctrl.signerRepo.GetByUser(userId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	slog.Info("Signer get by user successful", "count", len(signers))

	return response.SendSuccess(c, "Signer fetched", signers)
}
