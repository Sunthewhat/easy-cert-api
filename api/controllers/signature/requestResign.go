package signature_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	signermodel "github.com/sunthewhat/easy-cert-api/api/model/signerModel"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func RequestResign(c *fiber.Ctx) error {
	signatureId := c.Params("signatureId")

	signature, err := signaturemodel.GetById(signatureId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	signer, err := signermodel.GetById(signature.SignerID)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	cert, err := certificatemodel.GetById(signature.CertificateID)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	err = util.SendSignatureRequestMail(signer.Email, signer.DisplayName, cert.ID, cert.Name)

	if err != nil {
		slog.Error("Failed to send new signature request mail", "error", err, "signatureId", signatureId)
		return response.SendInternalError(c, err)
	}

	err = signaturemodel.UpdateAfterRequestResign(signatureId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	return response.SendSuccess(c, "Request resign certificate successfully")
}
