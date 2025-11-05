package signature_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/api/middleware"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

type SignerDataResponse struct {
	Signature *SignatureResponseDTO `json:"signature"`
	Signer    *SignerInfo           `json:"signer"`
}

type SignerInfo struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

func (ctrl *SignatureController) GetSignerData(c *fiber.Ctx) error {
	certificateId := c.Params("certificateId")

	if certificateId == "" {
		return response.SendFailed(c, "Certificate ID is required")
	}

	// Get user email from context
	userEmail, ok := middleware.GetUserFromContext(c)
	if !ok {
		slog.Error("GetSignerData: Failed to get user from context")
		return response.SendError(c, "Failed to read user")
	}

	cert, err := ctrl.certificateRepo.GetById(certificateId)
	if err != nil {
		slog.Error("GetSignerData: Error getting certificate", "certId", certificateId)
		return response.SendError(c, "Certificate not found")
	}

	slog.Info("Requesting Signer Data", "certId", certificateId, "user", userEmail)

	// Get signer by email
	signer, err := ctrl.signerRepo.GetByEmail(userEmail, cert.UserID)
	if err != nil {
		slog.Error("GetSignerData: Error fetching signer", "error", err, "email", userEmail)
		return response.SendInternalError(c, err)
	}

	if signer == nil {
		return response.SendFailed(c, "Signer not found")
	}

	slog.Info("Found Signer", "signerId", signer.ID)

	// Get signature by certificate ID and signer ID
	signature, err := ctrl.signatureRepo.GetByCertificateAndSignerId(certificateId, signer.ID)
	if err != nil {
		slog.Error("GetSignerData: Error fetching signature", "error", err, "certificateId", certificateId, "signerId", signer.ID)
		return response.SendInternalError(c, err)
	}

	if signature == nil {
		return response.SendFailed(c, "Signature not found for this certificate")
	}

	// Build response
	responseData := SignerDataResponse{
		Signature: &SignatureResponseDTO{
			ID:            signature.ID,
			SignerID:      signature.SignerID,
			CertificateID: signature.CertificateID,
			CreatedAt:     signature.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			IsSigned:      signature.IsSigned,
			CreatedBy:     signature.CreatedBy,
			IsRequested:   signature.IsRequested,
			LastRequest:   signature.LastRequest.Format("2006-01-02T15:04:05Z07:00"),
		},
		Signer: &SignerInfo{
			ID:          signer.ID,
			Email:       signer.Email,
			DisplayName: signer.DisplayName,
			CreatedAt:   signer.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	return response.SendSuccess(c, "Signer data retrieved successfully", responseData)
}
