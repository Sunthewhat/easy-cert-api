package participant_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

func GetByCert(c *fiber.Ctx) error {
	certId := c.Params("certId")
	status := c.Query("status")

	if certId == "" {
		slog.Warn("Request get Participant with empty certificate ID")
		return response.SendFailed(c, "Certificate ID is required")
	}

	cert, err := certificatemodel.GetById(certId)

	if err != nil {
		slog.Error("Get Participant by ID failed", "error", err, "certId", certId)
		return response.SendInternalError(c, err)
	}

	if cert == nil {
		slog.Warn("Get Participant with non-existing certificate", "certId", certId)
		return response.SendFailed(c, "Certificate not found")
	}

	participants, err := participantmodel.GetParticipantsByCertId(certId)

	if err != nil {
		slog.Error("Get participant Error", "error", err)
		return response.SendInternalError(c, err)
	}

	// Initialize empty slice to avoid returning null
	if participants == nil {
		participants = []*participantmodel.CombinedParticipant{}
	}

	// Filter participants based on status query parameter
	if status == "distributed" {
		distributedParticipants := []*participantmodel.CombinedParticipant{}
		for _, participant := range participants {
			if participant.IsDistributed {
				distributedParticipants = append(distributedParticipants, participant)
			}
		}
		return response.SendSuccess(c, "Distributed Participants Fetched!", distributedParticipants)
	}

	return response.SendSuccess(c, "Participant Fetched!", participants)
}
