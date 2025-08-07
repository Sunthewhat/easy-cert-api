package response

type BaseResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Data    any    `json:"data"`
}

func Success(msg string, data ...any) *BaseResponse {
	var responseData any = nil

	if len(data) > 0 {
		responseData = data[0]
	}

	return &BaseResponse{
		Success: true,
		Msg:     msg,
		Data:    responseData,
	}
}
