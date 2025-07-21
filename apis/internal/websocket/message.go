package websocket

// TODO: how do I format this?
type Message struct {
	Type int    `json:"type"`
	Body string `json:"body"`
}
