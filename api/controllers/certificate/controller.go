package certificate_controller

import (
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
)

// CertificateController handles certificate-related HTTP requests
type CertificateController struct {
	certRepo      *certificatemodel.CertificateRepository
	signatureRepo *signaturemodel.SignatureRepository
}

// NewCertificateController creates a new certificate controller with injected dependencies
func NewCertificateController(certRepo *certificatemodel.CertificateRepository, signatureRepo *signaturemodel.SignatureRepository) *CertificateController {
	return &CertificateController{
		certRepo:      certRepo,
		signatureRepo: signatureRepo,
	}
}
