package response

func Error(msg string) *BaseResponse {
	return &BaseResponse{
		Success: false,
		Msg:     msg,
		Data:    nil,
	}
}
