package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opisvigilant/futura/apis/internal/config"
	workflowsignals "github.com/opisvigilant/futura/apis/internal/workflow/signals"
	"github.com/rs/zerolog/log"

	"github.com/opisvigilant/futura/go-lib/stream"
)

const FetchClustersSSEEventName = "fetch_clusters_result"

type SSEController struct {
	js stream.Stream
}

func NewSSEController(config *config.Nats) (*SSEController, error) {
	js, err := stream.NewNATSJetstreamClient(context.Background(), config.Servers)
	if err != nil {
		return nil, err
	}
	return &SSEController{
		js: js,
	}, nil
}

func (s *SSEController) FetchClustersResultHandler(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		http.Error(c.Writer, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create a channel to receive messages
	msgCh := make(chan ConnectUpdate, 64)

	if err := s.js.Subscribe(workflowsignals.NatsWorkflowFetchClusterTopic, func(msg stream.Message) {
		var update ConnectUpdate
		if err := json.Unmarshal(msg.Data(), &update); err != nil {
			log.Logger.Error().Err(err).Msg("invalid NATS message")
			return
		}
		log.Logger.Debug().Interface("message", update).Msg("read message from queue")

		if err := msg.Ack(); err != nil {
			log.Logger.Error().Err(err).Msgf("failed to ACK message in topic `%s`", workflowsignals.NatsWorkflowFetchClusterTopic)
		}

		msgCh <- update
	}); err != nil {
		log.Logger.Error().Err(err).Msgf("failed to subscribe to topic `%s`", workflowsignals.NatsWorkflowFetchClusterTopic)
	}

	// Heartbeat ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Loop and stream messages
	for {
		select {
		case msg := <-msgCh:
			log.Logger.Debug().Interface("message", msg).Msg("stream message")

			b, err := json.Marshal(msg)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to marshal message to JSON")
				continue
			}
			c.SSEvent(FetchClustersSSEEventName, b)
			flusher.Flush()
		case <-ticker.C:
			// Send a comment to keep the connection alive
			c.Writer.Write([]byte(": ping\n\n"))
			flusher.Flush()
		case <-c.Request.Context().Done():
			// Client disconnected
			return
		}
	}
}

type ConnectUpdate struct {
	ConnectID uint   `json:"connect_id"`
	Status    string `json:"status"` // "started", "succeeded", "failed"
	Message   string `json:"message"`
}
