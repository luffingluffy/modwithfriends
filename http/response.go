package http

type standardResponse struct {
	Message string `json:"message"`
}

func newStandardResponse(msg string) standardResponse {
	return standardResponse{msg}
}
