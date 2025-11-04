package participant_controller

import (
	"github.com/sunthewhat/easy-cert-api/common"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func GetValidationDataByParticipantId(c *fiber.Ctx) error {
	participantId := c.Params("participantId")

	if participantId == "" {
		slog.Warn("Request validation without participant id")
		return response.SendFailed(c, "Participant Id is missing")
	}

	participant, err := participantmodel.GetParticipantsById(participantId)
	if err != nil {
		return response.SendInternalError(c, err)
	}

	certRepo := certificatemodel.NewCertificateRepository(common.Gorm)
	certificate, err := certRepo.GetById(participant.CertificateID)
	if err != nil {
		return response.SendInternalError(c, err)
	}

	if !participant.IsDownloaded && participant.EmailStatus != "success" {
		return response.SendFailed(c, "Participant not found")
	}

	return response.SendSuccess(c, "Participant data fetched", fiber.Map{
		"certificate": certificate,
		"participant": participant,
	})
}
