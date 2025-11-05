package certificate_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func (ctrl *CertificateController) DistributeByMail(c *fiber.Ctx) error {
	certId := c.Params("certId")
	emailField := c.Query("email")

	if emailField == "" {
		return response.SendFailed(c, "Missing email field")
	}

	cert, err := ctrl.certRepo.GetById(certId)
	if err != nil {
		slog.Error("Certificate Controller Distribute by Mail Error", "error", err)
		return response.SendInternalError(c, err)
	}

	if cert == nil {
		slog.Warn("Distribute By Mail with non-existing certificate", "certId", certId)
		return response.SendFailed(c, "Certificate not exist")
	}

	participants, err := ctrl.participantRepo.GetParticipantsByCertId(certId)
	if err != nil {
		slog.Error("Distribute By Mail Get participant by certId Error", "error", err)
		return response.SendInternalError(c, err)
	}

	var successResults []map[string]string
	var failedResults []map[string]string
	var skippedResults []map[string]string

	for _, participant := range participants {
		participantInfo := map[string]string{
			"participant_id": participant.ID,
		}

		// Skip if email was already sent successfully
		if participant.EmailStatus == "success" {
			participantInfo["status"] = "skipped"
			participantInfo["reason"] = "Email already sent successfully"
			skippedResults = append(skippedResults, participantInfo)
			slog.Info("Skipping participant - email already sent",
				"certId", certId,
				"participantId", participant.ID,
				"email_status", participant.EmailStatus)
			continue
		}

		if participant.CertificateURL == "" {
			participantInfo["error"] = "Certificate URL not found"
			failedResults = append(failedResults, participantInfo)
			slog.Error("Attempt to send mail without certificate url", "certId", certId, "participantId", participant.ID)
			// Update email status to failed
			ctrl.participantRepo.UpdateEmailStatus(participant.ID, "failed")
			continue
		}

		// Extract email from DynamicData using the emailField parameter
		emailValue, exists := participant.DynamicData[emailField]
		if !exists {
			participantInfo["error"] = "Email field not found in participant data"
			failedResults = append(failedResults, participantInfo)
			slog.Warn("Email field not found in participant data",
				"certId", certId,
				"participantId", participant.ID,
				"emailField", emailField)
			// Update email status to failed
			ctrl.participantRepo.UpdateEmailStatus(participant.ID, "failed")
			continue
		}

		// Convert to string
		email, ok := emailValue.(string)
		if !ok {
			participantInfo["error"] = "Email field is not a string"
			failedResults = append(failedResults, participantInfo)
			slog.Warn("Email field is not a string",
				"certId", certId,
				"participantId", participant.ID,
				"emailField", emailField,
				"emailValue", emailValue)
			// Update email status to failed
			ctrl.participantRepo.UpdateEmailStatus(participant.ID, "failed")
			continue
		}

		if email == "" {
			participantInfo["error"] = "Empty email address"
			failedResults = append(failedResults, participantInfo)
			slog.Warn("Empty email address",
				"certId", certId,
				"participantId", participant.ID)
			// Update email status to failed
			ctrl.participantRepo.UpdateEmailStatus(participant.ID, "failed")
			continue
		}

		participantInfo["email"] = email

		err := util.SendMail(email, participant.CertificateURL)
		if err != nil {
			participantInfo["error"] = err.Error()
			failedResults = append(failedResults, participantInfo)
			slog.Error("Failed to send mail to participant",
				"error", err,
				"certId", certId,
				"participantId", participant.ID,
				"email", email)
			// Update email status to failed
			ctrl.participantRepo.UpdateEmailStatus(participant.ID, "failed")
		} else {
			// Update email status to success
			err := ctrl.participantRepo.UpdateEmailStatus(participant.ID, "success")
			if err != nil {
				slog.Warn("Failed to update email status to success",
					"error", err,
					"participantId", participant.ID)
			}

			successResults = append(successResults, participantInfo)
			slog.Info("Mail sent successfully",
				"certId", certId,
				"participantId", participant.ID,
				"email", email)
		}
	}

	// Prepare response data
	responseData := map[string]any{
		"total_participants": len(participants),
		"success_count":      len(successResults),
		"failed_count":       len(failedResults),
		"skipped_count":      len(skippedResults),
		"success_results":    successResults,
		"failed_results":     failedResults,
		"skipped_results":    skippedResults,
	}

	return response.SendSuccess(c, "Mail distribution completed", responseData)
}

// ResendParticipantMail resends certificate email to a specific participant by their ID
func (ctrl *CertificateController) ResendParticipantMail(c *fiber.Ctx) error {
	participantId := c.Params("participantId")

	if participantId == "" {
		return response.SendFailed(c, "Participant ID is required")
	}

	// Get participant by ID
	participant, err := ctrl.participantRepo.GetParticipantsById(participantId)
	if err != nil {
		slog.Error("Resend Participant Mail: Error getting participant", "error", err, "participantId", participantId)
		return response.SendInternalError(c, err)
	}

	if participant == nil {
		slog.Warn("Resend Participant Mail: Participant not found", "participantId", participantId)
		return response.SendFailed(c, "Participant not found")
	}

	// Check if certificate URL exists
	if participant.CertificateURL == "" {
		slog.Error("Resend Participant Mail: Certificate URL not found", "participantId", participantId)
		ctrl.participantRepo.UpdateEmailStatus(participantId, "failed")
		return response.SendFailed(c, "Certificate URL not found for this participant")
	}

	// Extract email from DynamicData using the emailField parameter
	emailValue, exists := participant.DynamicData["email"]
	if !exists {
		slog.Warn("Resend Participant Mail: Email field not found in participant data",
			"participantId", participantId)
		ctrl.participantRepo.UpdateEmailStatus(participantId, "failed")
		return response.SendFailed(c, "Email field not found in participant data")
	}

	// Convert to string
	email, ok := emailValue.(string)
	if !ok {
		slog.Warn("Resend Participant Mail: Email field is not a string",
			"participantId", participantId,
			"emailValue", emailValue)
		ctrl.participantRepo.UpdateEmailStatus(participantId, "failed")
		return response.SendFailed(c, "Email field is not a valid string")
	}

	if email == "" {
		slog.Warn("Resend Participant Mail: Empty email address", "participantId", participantId)
		ctrl.participantRepo.UpdateEmailStatus(participantId, "failed")
		return response.SendFailed(c, "Empty email address")
	}

	// Send email
	err = util.SendMail(email, participant.CertificateURL)
	if err != nil {
		slog.Error("Resend Participant Mail: Failed to send email",
			"error", err,
			"participantId", participantId,
			"email", email)
		ctrl.participantRepo.UpdateEmailStatus(participantId, "failed")
		return response.SendError(c, "Failed to send email: "+err.Error())
	}

	// Update email status to success
	err = ctrl.participantRepo.UpdateEmailStatus(participantId, "success")
	if err != nil {
		slog.Warn("Resend Participant Mail: Failed to update email status",
			"error", err,
			"participantId", participantId)
		// Don't fail the request - email was sent successfully
	}

	slog.Info("Resend Participant Mail: Email sent successfully",
		"participantId", participantId,
		"email", email)

	// Prepare response data
	responseData := map[string]any{
		"participant_id":  participant.ID,
		"email":           email,
		"email_status":    "success",
		"certificate_url": participant.CertificateURL,
		"certificate_id":  participant.CertificateID,
	}

	return response.SendSuccess(c, "Email sent successfully", responseData)
}
