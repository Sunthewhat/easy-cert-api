package signature_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func (ctrl *SignatureController) GetSignatureImage(c *fiber.Ctx) error {
	certificateId := c.Params("certificateId")
	signerId := c.Params("signerId")

	userId, status := middleware.GetUserFromContext(c)

	if !status {
		slog.Error("Signature Get GetUserId failed")
		return response.SendError(c, "Failed to read user")
	}

	signature, err := ctrl.signatureRepo.GetByCertificateAndSignerId(certificateId, signerId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	if signature == nil {
		slog.Warn("Get non-existing signature")
		return response.SendFailed(c, "Signature not found")
	}

	if signature.SignerID != userId && signature.CreatedBy != userId {
		slog.Error("Invalid credential try to get signature")
		return response.SendFailed(c, "You do not owned this signature")
	}

	// Check if signature has been signed
	if !signature.IsSigned {
		return response.SendFailed(c, "Signature has not been uploaded yet")
	}

	// Decrypt the signature image
	decryptedImage, err := util.DecryptData(signature.Signature, *common.Config.EncryptionKey)
	if err != nil {
		slog.Error("Failed to decrypt signature", "error", err, "id", signature.ID)
		return response.SendError(c, "Failed to decrypt signature")
	}

	// Return image with appropriate content type
	c.Set("Content-Type", "image/png")
	return c.Send(decryptedImage)
}
