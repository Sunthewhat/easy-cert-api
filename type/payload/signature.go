package payload

type CreateSignaturePayload struct {
	CertificateId string `json:"certificate_id" validate:"required"`
	SignerId      string `json:"signer_id" validate:"required"`
}
