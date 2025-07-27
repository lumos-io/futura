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
	rs stream.Stream
}

func NewSSEController(config *config.Redis) (*SSEController, error) {
	js, err := stream.NewRedisStreamClient(config.Servers)
	if err != nil {
		return nil, err
	}
	return &SSEController{
		rs: js,
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
	msgCh := make(chan *workflowsignals.WorkflowFetchClustersStatusSignal)

	if err := s.rs.Subscribe(context.Background(), workflowsignals.NatsWorkflowFetchClusterTopic, func(msg stream.Message, ack func() error) {
		var result *workflowsignals.WorkflowFetchClustersStatusSignal
		if err := json.Unmarshal(msg.Data(), &result); err != nil {
			log.Logger.Error().Err(err).Msg("invalid NATS message")
			return
		}
		// push message to the channel
		msgCh <- result
		if err := ack(); err != nil {
			log.Logger.Error().Err(err).Msg("failed to ack message")
		}
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
