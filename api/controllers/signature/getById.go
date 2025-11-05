package signature_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

type SignatureResponseDTO struct {
	ID            string `json:"id"`
	SignerID      string `json:"signer_id"`
	CertificateID string `json:"certificate_id"`
	CreatedAt     string `json:"created_at"`
	IsSigned      bool   `json:"is_signed"`
	CreatedBy     string `json:"created_by"`
	IsRequested   bool   `json:"is_requested"`
	LastRequest   string `json:"last_request"`
}

func (ctrl *SignatureController) GetById(c *fiber.Ctx) error {
	id := c.Params("id")

	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Signature ID is required",
		})
	}

	signature, err := ctrl.signatureRepo.GetById(id)
	if err != nil {
		slog.Error("GetById Error", "error", err, "id", id)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Failed to get signature",
		})
	}

	if signature == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "Signature not found",
		})
	}

	// Map to DTO without the encrypted signature data
	response := SignatureResponseDTO{
		ID:            signature.ID,
		SignerID:      signature.SignerID,
		CertificateID: signature.CertificateID,
		CreatedAt:     signature.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		IsSigned:      signature.IsSigned,
		CreatedBy:     signature.CreatedBy,
		IsRequested:   signature.IsRequested,
		LastRequest:   signature.LastRequest.Format("2006-01-02T15:04:05Z07:00"),
	}

	return c.JSON(response)
}
