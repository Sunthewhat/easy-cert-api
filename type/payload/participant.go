package payload

type AddParticipantPayload struct {
	Participants []map[string]any `json:"participants" validate:"required"`
}
