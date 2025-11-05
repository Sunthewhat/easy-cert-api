package certificate_controller

import (
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	participantmodel "github.com/sunthewhat/easy-cert-api/api/model/participantModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
)

// CertificateController handles certificate-related HTTP requests
type CertificateController struct {
	certRepo        certificatemodel.ICertificateRepository
	signatureRepo   signaturemodel.ISignatureRepository
	participantRepo participantmodel.IParticipantRepository
}

// NewCertificateController creates a new certificate controller with injected dependencies
func NewCertificateController(
	certRepo certificatemodel.ICertificateRepository,
	signatureRepo signaturemodel.ISignatureRepository,
	participantRepo participantmodel.IParticipantRepository,
) *CertificateController {
	return &CertificateController{
		certRepo:        certRepo,
		signatureRepo:   signatureRepo,
		participantRepo: participantRepo,
	}
}
