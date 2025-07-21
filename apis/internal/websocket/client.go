package websocket

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type Connection interface {
	WriteJSON(v any) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
}

type Client struct {
	ID        uuid.UUID
	ConnectID uint
	Conn      Connection
	Pool      *Pool
}

func (c *Client) Read() {
	defer func() {
		c.Pool.Unregister <- c
		if err := c.Conn.Close(); err != nil {
			log.Logger.Error().Err(err).Msg("failed to close the connection")
			return
		}
	}()

	for {
		messageType, p, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Logger.Info().Err(err).Msg("client closed connection")
			} else {
				log.Logger.Error().Err(err).Msg("Unexpected websocket error")
			}
			break
		}
		message := Message{Type: messageType, Body: string(p)}
		c.Pool.Broadcast <- message
	}
}
