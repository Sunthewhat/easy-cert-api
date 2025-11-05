package certificate_controller

import (
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
)

// CertificateController handles certificate-related HTTP requests
type CertificateController struct {
	certRepo        *certificatemodel.CertificateRepository
	signatureRepo   *signaturemodel.SignatureRepository
	participantRepo *participantmodel.ParticipantRepository
}

// NewCertificateController creates a new certificate controller with injected dependencies
func NewCertificateController(
	certRepo *certificatemodel.CertificateRepository,
	signatureRepo *signaturemodel.SignatureRepository,
	participantRepo *participantmodel.ParticipantRepository,
) *CertificateController {
	return &CertificateController{
		certRepo:        certRepo,
		signatureRepo:   signatureRepo,
		participantRepo: participantRepo,
	}
}
