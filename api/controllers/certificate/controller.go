package certificate_controller

import (
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
)

// CertificateController handles certificate-related HTTP requests
type CertificateController struct {
	certRepo *certificatemodel.CertificateRepository
}

// NewCertificateController creates a new certificate controller with injected dependencies
func NewCertificateController(certRepo *certificatemodel.CertificateRepository) *CertificateController {
	return &CertificateController{
		certRepo: certRepo,
	}
}
