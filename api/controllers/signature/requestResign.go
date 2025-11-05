package signature_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func (ctrl *SignatureController) RequestResign(c *fiber.Ctx) error {
	signatureId := c.Params("signatureId")

	signature, err := ctrl.signatureRepo.GetById(signatureId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	signer, err := ctrl.signerRepo.GetById(signature.SignerID)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	cert, err := ctrl.certificateRepo.GetById(signature.CertificateID)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	err = util.SendSignatureRequestMail(signer.Email, signer.DisplayName, cert.ID, cert.Name)

	if err != nil {
		slog.Error("Failed to send new signature request mail", "error", err, "signatureId", signatureId)
		return response.SendInternalError(c, err)
	}

	err = ctrl.signatureRepo.UpdateAfterRequestResign(signatureId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	return response.SendSuccess(c, "Request resign certificate successfully")
}
