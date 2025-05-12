package response

import "github.com/bsthun/gut"

type ErrorResponse struct {
	Success bool    `json:"success"`
	Message *string `json:"message,omitempty"`
}

func Error(msg any) *ErrorResponse {
	if message, ok := msg.(string); ok {

		return &ErrorResponse{
			Success: false,
			Message: &message,
		}
	}
	return &ErrorResponse{
		Success: false,
		Message: gut.Ptr("Unknown Error"),
	}
}
