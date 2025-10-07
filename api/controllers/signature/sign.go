package signature_controller

import (
	"io"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func Sign(c *fiber.Ctx) error {
	// 1. Get signature ID from URL params
	signatureId := c.Params("id")
	if signatureId == "" {
		return response.SendFailed(c, "Signature ID is required")
	}

	// 2. Receive signature image file
	fileHeader, err := c.FormFile("signature_image")
	if err != nil {
		slog.Error("No signature image provided", "error", err)
		return response.SendFailed(c, "Signature image is required")
	}

	// 3. Validate file size (limit to 10MB for signature images)
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if fileHeader.Size > maxSize {
		return response.SendFailed(c, "Signature image too large (max 5MB)")
	}

	// 4. Read file contents
	file, err := fileHeader.Open()
	if err != nil {
		slog.Error("Failed to open signature image", "error", err)
		return response.SendInternalError(c, err)
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		slog.Error("Failed to read signature image", "error", err)
		return response.SendInternalError(c, err)
	}

	// 5. Encrypt the signature image
	encryptedSignature, err := util.EncryptData(imageData, *common.Config.EncryptionKey)
	if err != nil {
		slog.Error("Failed to encrypt signature", "error", err)
		return response.SendError(c, "Failed to encrypt signature")
	}

	// 6. Update the signature record with encrypted data and mark as signed
	updatedSignature, err := signaturemodel.UpdateSignature(signatureId, encryptedSignature)
	if err != nil {
		return response.SendInternalError(c, err)
	}

	// 7. Return success response
	return response.SendSuccess(c, "Signature uploaded and encrypted successfully", fiber.Map{
		"signature_id":   updatedSignature.ID,
		"signer_id":      updatedSignature.SignerID,
		"certificate_id": updatedSignature.CertificateID,
		"is_signed":      updatedSignature.IsSigned,
		"created_at":     updatedSignature.CreatedAt,
	})
}
