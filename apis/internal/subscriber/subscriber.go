package subscriber

import (
	"context"
	"encoding/json"

	"github.com/opisvigilant/futura/apis/internal/websocket"
	"github.com/opisvigilant/futura/go-lib/stream"
	"github.com/rs/zerolog/log"
)

type ConnectUpdate struct {
	ConnectID uint   `json:"connect_id"`
	Status    string `json:"status"` // "started", "succeeded", "failed"
	Message   string `json:"message"`
}

func StartConnectSubscribers(pool *websocket.Pool, natsURL []string, topics []string) {
	js, err := stream.NewNATSJetstreamClient(context.Background(), natsURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to Stream")
		return
	}

	for _, topic := range topics {
		if err := js.Subscribe(topic, func(msg stream.Message) {
			var update ConnectUpdate
			if err := json.Unmarshal(msg.Data(), &update); err != nil {
				log.Error().Err(err).Msg("Invalid NATS message")
				return
			}

			message := websocket.Message{
				Type: 1,
				Body: update.Message,
			}

			for client := range pool.Clients {
				if client.ConnectID == update.ConnectID {
					client.Conn.WriteJSON(message)
				}
			}
		}); err != nil {
			log.Error().Err(err).Msg("Invalid NATS message")
		}
	}
}
