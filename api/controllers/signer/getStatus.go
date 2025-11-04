package signer_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	"github.com/sunthewhat/easy-cert-api/common"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	signermodel "github.com/sunthewhat/easy-cert-api/api/model/signerModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

type signatureWithSignerData struct {
	ID          string `json:"id"`
	IsSigned    bool   `json:"is_signed"`
	IsRequested bool   `json:"is_requested"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func GetStatus(c *fiber.Ctx) error {
	userId, success := middleware.GetUserFromContext(c)

	if !success {
		slog.Error("Get Signature Status User not found from context")
		return response.SendUnauthorized(c, "User context failed")
	}

	certId := c.Params("certId")

	certRepo := certificatemodel.NewCertificateRepository(common.Gorm)
	cert, err := certRepo.GetById(certId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	if cert.UserID != userId {
		slog.Warn("User try to access certificate they not own", "user", userId, "certId", certId)
		return response.SendUnauthorized(c, "You did not own this certificate")
	}

	signatures, err := signaturemodel.GetSignaturesByCertificate(certId)

	if err != nil {
		return response.SendInternalError(c, err)
	}

	var signatureDataResponse []*signatureWithSignerData

	for _, sig := range signatures {
		signer, err := signermodel.GetById(sig.SignerID)
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
