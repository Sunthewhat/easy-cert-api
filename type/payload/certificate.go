package payload

type UpdateCertificatePayload struct {
	Name   string `json:"name"`
	Design string `json:"design"`
}

type CreateCertificatePayload struct {
	Name   string `json:"name" validate:"required"`
	Design string `json:"design" validate:"required"`
}