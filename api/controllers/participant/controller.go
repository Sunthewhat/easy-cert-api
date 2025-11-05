package participant_controller

import (
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
)

// ParticipantController handles participant-related HTTP requests
type ParticipantController struct {
	participantRepo *participantmodel.ParticipantRepository
	certificateRepo *certificatemodel.CertificateRepository
}

// NewParticipantController creates a new participant controller with injected dependencies
func NewParticipantController(
	participantRepo *participantmodel.ParticipantRepository,
	certificateRepo *certificatemodel.CertificateRepository,
) *ParticipantController {
	return &ParticipantController{
		participantRepo: participantRepo,
		certificateRepo: certificateRepo,
	}
}
