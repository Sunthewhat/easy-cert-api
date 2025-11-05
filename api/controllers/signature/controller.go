package signature_controller

import (
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	signermodel "github.com/sunthewhat/easy-cert-api/api/model/signerModel"
)

// SignatureController handles signature-related HTTP requests
type SignatureController struct {
	signatureRepo   *signaturemodel.SignatureRepository
	certificateRepo *certificatemodel.CertificateRepository
	signerRepo      *signermodel.SignerRepository
}

// NewSignatureController creates a new signature controller with injected dependencies
func NewSignatureController(
	signatureRepo *signaturemodel.SignatureRepository,
	certificateRepo *certificatemodel.CertificateRepository,
	signerRepo *signermodel.SignerRepository,
) *SignatureController {
	return &SignatureController{
		signatureRepo:   signatureRepo,
		certificateRepo: certificateRepo,
		signerRepo:      signerRepo,
	}
}
