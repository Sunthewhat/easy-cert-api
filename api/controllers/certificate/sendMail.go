package certificate_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/common/util"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func DistributeByMail(c *fiber.Ctx) error {
	certId := c.Params("certId")
	emailField := c.Query("email")

	if emailField == "" {
		return response.SendFailed(c, "Missing email field")
	}

	cert, err := certificatemodel.GetById(certId)
	if err != nil {
		slog.Error("Certificate Controller Distribute by Mail Error", "error", err)
		return response.SendInternalError(c, err)
	}

	if cert == nil {
		slog.Warn("Distribute By Mail with non-existing certificate", "certId", certId)
		return response.SendFailed(c, "Certificate not exist")
	}

	participants, err := participantmodel.GetParticipantsByCertId(certId)
	if err != nil {
		slog.Error("Distribute By Mail Get participant by certId Error", "error", err)
		return response.SendInternalError(c, err)
	}

	var successResults []map[string]string
	var failedResults []map[string]string

	for _, participant := range participants {
		participantInfo := map[string]string{
			"participant_id": participant.ID,
		}

		if participant.CertificateURL == "" {
			participantInfo["error"] = "Certificate URL not found"
			failedResults = append(failedResults, participantInfo)
			slog.Error("Attempt to send mail without certificate url", "certId", certId, "participantId", participant.ID)
			// Update email status to failed
			participantmodel.UpdateEmailStatus(participant.ID, "failed")
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
			participantmodel.UpdateEmailStatus(participant.ID, "failed")
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
			participantmodel.UpdateEmailStatus(participant.ID, "failed")
			continue
		}

		if email == "" {
			participantInfo["error"] = "Empty email address"
			failedResults = append(failedResults, participantInfo)
			slog.Warn("Empty email address",
				"certId", certId,
				"participantId", participant.ID)
			// Update email status to failed
			participantmodel.UpdateEmailStatus(participant.ID, "failed")
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
			participantmodel.UpdateEmailStatus(participant.ID, "failed")
		} else {
			// Update email status to success
			err := participantmodel.UpdateEmailStatus(participant.ID, "success")
			if err != nil {
				slog.Warn("Failed to update email status to success",
					"error", err,
					"participantId", participant.ID)
			}

			err = participantmodel.MarkParticipantAsDistributed(participant.ID)
			if err != nil {
				participantInfo["error"] = err.Error()
				failedResults = append(failedResults, participantInfo)
				continue
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
		"success_results":    successResults,
		"failed_results":     failedResults,
	}

	return response.SendSuccess(c, "Mail distribution completed", responseData)
}
