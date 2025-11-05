package signer_controller

import (
	certificatemodel "github.com/sunthewhat/easy-cert-api/api/model/certificateModel"
	signaturemodel "github.com/sunthewhat/easy-cert-api/api/model/signatureModel"
	signermodel "github.com/sunthewhat/easy-cert-api/api/model/signerModel"
)

// SignerController handles signer-related HTTP requests
type SignerController struct {
	signerRepo      signermodel.ISignerRepository
	signatureRepo   signaturemodel.ISignatureRepository
	certificateRepo certificatemodel.ICertificateRepository
}

// NewSignerController creates a new signer controller with injected dependencies
func NewSignerController(
	signerRepo signermodel.ISignerRepository,
	signatureRepo signaturemodel.ISignatureRepository,
	certificateRepo certificatemodel.ICertificateRepository,
) *SignerController {
	return &SignerController{
		signerRepo:      signerRepo,
		signatureRepo:   signatureRepo,
		certificateRepo: certificateRepo,
	}
}
