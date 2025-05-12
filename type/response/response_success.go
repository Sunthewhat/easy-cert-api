package response

type SuccessResponse struct {
	Success bool    `json:"success"`
	Message *string `json:"message,omitempty"`
	Data    any     `json:"data,omitempty"`
}

func Success(msg any, data ...any) *SuccessResponse {
	// Case 1: msg is a string (message)
	if message, ok := msg.(string); ok {
		// Create response with message
		response := &SuccessResponse{
			Success: true,
			Message: &message,
		}

		// If data is provided, add it to the response
		if len(data) > 0 {
			response.Data = data[0]
		}

		return response
	}

	// Case 2: msg is not a string (it's the data)
	return &SuccessResponse{
		Success: true,
		Data:    msg,
	}
}
