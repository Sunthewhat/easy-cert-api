package signer_controller

import (
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	signermodel "github.com/sunthewhat/easy-cert-api/api/model/signerModel"
)

// SignerController handles signer-related HTTP requests
type SignerController struct {
	signerRepo      *signermodel.SignerRepository
	signatureRepo   *signaturemodel.SignatureRepository
	certificateRepo *certificatemodel.CertificateRepository
}

// NewSignerController creates a new signer controller with injected dependencies
func NewSignerController(
	signerRepo *signermodel.SignerRepository,
	signatureRepo *signaturemodel.SignatureRepository,
	certificateRepo *certificatemodel.CertificateRepository,
) *SignerController {
	return &SignerController{
		signerRepo:      signerRepo,
		signatureRepo:   signatureRepo,
		certificateRepo: certificateRepo,
	}
}
