package signature_controller

import (
	"io"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
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

	// 7. Check if all signatures are complete for this certificate
	allComplete, checkErr := signaturemodel.AreAllSignaturesComplete(updatedSignature.CertificateID)
	if checkErr != nil {
		slog.Error("Failed to check if all signatures complete", "error", checkErr, "certificateId", updatedSignature.CertificateID)
		// Don't fail the request - signature was uploaded successfully
	}

	// 8. If all signatures are complete, mark certificate as signed and notify owner
	if allComplete {
		certificate, certErr := certificatemodel.GetById(updatedSignature.CertificateID)
		if certErr != nil {
			slog.Error("Failed to get certificate for notification", "error", certErr, "certificateId", updatedSignature.CertificateID)
		} else if certificate != nil {
			// Mark certificate as fully signed
			markErr := certificatemodel.MarkAsSigned(certificate.ID)
			if markErr != nil {
				slog.Error("Failed to mark certificate as signed", "error", markErr, "certificateId", certificate.ID)
				// Don't fail the request - signature was uploaded successfully
			}

			// Send notification email to certificate owner
			notifyErr := util.SendAllSignaturesCompleteMail(certificate.UserID, certificate.Name, certificate.ID)
			if notifyErr != nil {
				slog.Error("Failed to send completion notification email", "error", notifyErr, "certificateId", certificate.ID, "owner", certificate.UserID)
				// Don't fail the request - signature was uploaded successfully
			} else {
				slog.Info("Certificate owner notified of completion", "certificateId", certificate.ID, "owner", certificate.UserID)
			}
		}
	}

	// 9. Return success response
	return response.SendSuccess(c, "Signature uploaded and encrypted successfully", fiber.Map{
		"signature_id":   updatedSignature.ID,
		"signer_id":      updatedSignature.SignerID,
		"certificate_id": updatedSignature.CertificateID,
		"is_signed":      updatedSignature.IsSigned,
		"created_at":     updatedSignature.CreatedAt,
		"all_complete":   allComplete,
	})
}
