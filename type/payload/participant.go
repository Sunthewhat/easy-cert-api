package payload

type AddParticipantPayload struct {
	Participants []map[string]any `json:"participants" validate:"required"`
}

type UpdateParticipantIsDistributed struct {
	Ids []string `json:"participantIds" validate:"required"`
}
