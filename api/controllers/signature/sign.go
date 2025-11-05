package signature_controller

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/internal/renderer"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func (ctrl *SignatureController) Sign(c *fiber.Ctx) error {
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
	updatedSignature, err := ctrl.signatureRepo.UpdateSignature(signatureId, encryptedSignature)
	if err != nil {
		return response.SendInternalError(c, err)
	}

	// 7. Check if all signatures are complete for this certificate
	allComplete, checkErr := ctrl.signatureRepo.AreAllSignaturesComplete(updatedSignature.CertificateID)
	if checkErr != nil {
		slog.Error("Failed to check if all signatures complete", "error", checkErr, "certificateId", updatedSignature.CertificateID)
		// Don't fail the request - signature was uploaded successfully
	}

	// 8. If all signatures are complete, mark certificate as signed and notify owner
	if allComplete {
		certificate, certErr := ctrl.certificateRepo.GetById(updatedSignature.CertificateID)
		if certErr != nil {
			slog.Error("Failed to get certificate for notification", "error", certErr, "certificateId", updatedSignature.CertificateID)
		} else if certificate != nil {
			// Mark certificate as fully signed
			markErr := ctrl.certificateRepo.MarkAsSigned(certificate.ID)
			if markErr != nil {
				slog.Error("Failed to mark certificate as signed", "error", markErr, "certificateId", certificate.ID)
				// Don't fail the request - signature was uploaded successfully
			}

			// Generate preview certificate with signatures and watermark
			previewPath := ""
			previewErr := func() error {
				// Get all participants
				participants, partErr := ctrl.participantRepo.GetParticipantsByCertId(certificate.ID)
				if partErr != nil {
					return partErr
				}

				if len(participants) == 0 {
					slog.Warn("No participants found for preview generation", "certificateId", certificate.ID)
					return nil // Skip preview generation
				}

				// Get all signatures and decrypt them
				allSignatures, sigErr := ctrl.signatureRepo.GetSignaturesByCertificate(certificate.ID)
				if sigErr != nil {
					return sigErr
				}

				// Decrypt signature images and create a map of signerId -> base64 image
				decryptedSignatures := make(map[string]string)
				for _, sig := range allSignatures {
					if sig.IsSigned && sig.Signature != "" {
						decryptedImage, decryptErr := util.DecryptData(sig.Signature, *common.Config.EncryptionKey)
						if decryptErr != nil {
							slog.Warn("Failed to decrypt signature for preview",
								"error", decryptErr,
								"cert_id", certificate.ID,
								"signer_id", sig.SignerID)
							continue
						}
						decryptedSignatures[sig.SignerID] = base64.StdEncoding.EncodeToString(decryptedImage)
					}
				}

				slog.Info("Decrypted signatures for preview",
					"cert_id", certificate.ID,
					"total_signatures", len(allSignatures),
					"decrypted_count", len(decryptedSignatures))

				// Convert participants to interface{} slice
				participantInterfaces := make([]any, len(participants))
				for i, p := range participants {
					participantInterfaces[i] = p
				}

				// Prepare certificate design
				certDesign := certificate.Design
				if *common.Config.Environment {
					certDesign = strings.ReplaceAll(
						certDesign,
						"http://easycert.sit.kmutt.ac.th",
						"http://backend:8000",
					)
				}

				// Convert certificate struct to map for renderer compatibility
				certMap := map[string]any{
					"id":     certificate.ID,
					"name":   certificate.Name,
					"design": certDesign,
				}

				// Initialize embedded renderer
				embeddedRenderer, rendererErr := renderer.NewEmbeddedRenderer()
				if rendererErr != nil {
					return rendererErr
				}
				defer embeddedRenderer.Close()

				// Create context with timeout
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()

				// Generate preview with watermark
				previewBytes, previewGenErr := embeddedRenderer.GeneratePreviewWithWatermark(ctx, certMap, participantInterfaces, decryptedSignatures, certificate.ID)
				if previewGenErr != nil {
					return previewGenErr
				}

				// Save preview to MinIO
				savedPath, saveErr := embeddedRenderer.SavePreviewToMinIO(previewBytes, certificate.ID)
				if saveErr != nil {
					return saveErr
				}

				previewPath = savedPath
				slog.Info("Preview generated and saved successfully", "certificateId", certificate.ID, "previewPath", previewPath)
				return nil
			}()

			if previewErr != nil {
				slog.Error("Failed to generate preview certificate", "error", previewErr, "certificateId", certificate.ID)
				// Don't fail the request - signature was uploaded successfully
			}

			// Send notification email to certificate owner with preview
			notifyErr := util.SendAllSignaturesCompleteMail(certificate.UserID, certificate.Name, certificate.ID, previewPath)
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
