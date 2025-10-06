package payload

type CreateSignerPayload struct {
	Email       string `json:"email" validate:"required"`
	DisplayName string `json:"display_name" validate:"required"`
}
