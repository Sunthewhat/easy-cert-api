package certificate_controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	"github.com/sunthewhat/easy-cert-api/type/response"
)

type responseStruct struct {
	IsSigned           bool `json:"is_signed"`
	IsGenerated        bool `json:"is_generated"`
	IsPartialGenerated bool `json:"is_partial_generated"`
}

func (ctrl *CertificateController) CheckGenerateStatus(c *fiber.Ctx) error {
	certificateId := c.Params("certificateId")

	cert, err := ctrl.certRepo.GetById(certificateId)

	if err != nil {
		slog.Error("Error getting certificate in Check Distribute Status controller", "error", err, "certId", certificateId)
		return response.SendInternalError(c, err)
	}

	if cert == nil {
		slog.Warn("CheckDistributeStatus trying to get non exisitng certificate", "certificateId", certificateId)
		return response.SendFailed(c, "certificate not found")
	}

	returnResponse := new(responseStruct)

	if !cert.IsSigned {
		notHaveSignature, err := signaturemodel.AreAllSignaturesComplete(certificateId)
		if err != nil {
			return response.SendInternalError(c, err)
		}

		if !notHaveSignature {
			returnResponse = &responseStruct{
				IsSigned:           false,
				IsGenerated:        false,
				IsPartialGenerated: false,
			}

			return response.SendSuccess(c, "Certificate is not signed", returnResponse)
		}
	}

	if !cert.IsDistributed {
		returnResponse = &responseStruct{
			IsSigned:           true,
			IsGenerated:        false,
			IsPartialGenerated: false,
		}
		return response.SendSuccess(c, "Certificate is not distributed", returnResponse)
	}

	participants, err := participantmodel.GetParticipantsByCertId(cert.ID)

	if err != nil {
		slog.Error("Error getting participants by certificate id in CheckDistributeStatus", "error", err, "certificateId", certificateId)
		return response.SendInternalError(c, err)
	}

	isPartialGenerated := false

	for _, p := range participants {
		if p.CertificateURL == "" {
			isPartialGenerated = true
			break
		}
	}

	returnResponse = &responseStruct{
		IsSigned:           true,
		IsGenerated:        true,
		IsPartialGenerated: isPartialGenerated,
	}

	return response.SendSuccess(c, "Certificate is distributed", returnResponse)
}
