package presenter

// MessageResponse is a simple { "message": "..." } envelope used by
// cache handlers, error paths, and middleware.
type MessageResponse struct {
	Message string `json:"message"`
}

// Message builds a simple message response.
func Message(msg string) MessageResponse {
	return MessageResponse{Message: msg}
}
