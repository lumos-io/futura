package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/opisvigilant/futura/apis/internal/config"
	"github.com/opisvigilant/futura/apis/internal/subscriber"
	"github.com/opisvigilant/futura/apis/internal/websocket"
	workflowsignals "github.com/opisvigilant/futura/apis/internal/workflow/signals"
	"github.com/opisvigilant/futura/apis/utils"
)

type WebsocketController struct {
	pool *websocket.Pool
}

func NewWebsocketController(config *config.Nats) *WebsocketController {
	pool := websocket.NewPool()
	go pool.Start()

	go subscriber.StartConnectSubscribers(pool, config.Servers, []string{
		workflowsignals.NatsWorkflowFetchClusterTopic,
	})

	return &WebsocketController{
		pool: pool,
	}
}

func (w *WebsocketController) Handler(c *gin.Context) {
	ws, err := websocket.Upgrade(c.Writer, c.Request)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "FAILED_WS_CONNECTION", "Could not upgrade to WebSocket")
		return
	}

	connectID, err := parseConnectID(c)
	if err != nil {
		return
	}

	client := &websocket.Client{
		ID:        uuid.New(),
		ConnectID: connectID,
		Conn:      ws,
		Pool:      w.pool,
	}

	w.pool.Register <- client
	client.Read()
}
