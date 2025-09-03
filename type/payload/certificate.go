package payload

type UpdateCertificatePayload struct {
	Name   string `json:"name"`
	Design string `json:"design"`
}

type CreateCertificatePayload struct {
	Name   string `json:"name" validate:"required"`
	Design string `json:"design" validate:"required"`
}

type renderCertificateResult struct {
	FilePath      string `json:"filePath"`
	ParticipantId string `json:"participantId"`
	Status        string `json:"status"`
}

type RenderCertificatePayload struct {
	Message     string                    `json:"message"`
	Results     []renderCertificateResult `json:"results"`
	ZipFilePath string                    `json:"zipFilePath"`
}

type RenderThumbnailPayload struct {
	Message       string `json:"message"`
	ThumbnailPath string `json:"thumbnailPath"`
}
