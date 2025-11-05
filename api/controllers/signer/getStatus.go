package signer_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

type signatureWithSignerData struct {
	ID          string `json:"id"`
	IsSigned    bool   `json:"is_signed"`
	IsRequested bool   `json:"is_requested"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func (ctrl *SignerController) GetStatus(c *fiber.Ctx) error {
	userId, success := middleware.GetUserFromContext(c)

	if !success {
		slog.Error("Get Signature Status User not found from context")
		return response.SendUnauthorized(c, "User context failed")
	}

	certId := c.Params("certId")

	cert, err := ctrl.certificateRepo.GetById(certId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	if cert.UserID != userId {
		slog.Warn("User try to access certificate they not own", "user", userId, "certId", certId)
		return response.SendUnauthorized(c, "You did not own this certificate")
	}

	signatures, err := ctrl.signatureRepo.GetSignaturesByCertificate(certId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	var signatureDataResponse []*signatureWithSignerData

	for _, sig := range signatures {
		signer, err := ctrl.signerRepo.GetById(sig.SignerID)
		if err != nil {
			slog.Error("Failed to get signer data from signature")
		} else {
			signatureDataResponse = append(signatureDataResponse, &signatureWithSignerData{
				ID:          sig.ID,
				IsSigned:    sig.IsSigned,
				IsRequested: sig.IsRequested,
				Email:       signer.Email,
				DisplayName: signer.DisplayName,
			})
		}
	}

	return response.SendSuccess(c, "Get signer status successfully", signatureDataResponse)
}
