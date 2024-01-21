package handlers

import (
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/l7_req"
	"github.com/opisvigilant/futura/watcher/internal/handlers/console"
	"github.com/opisvigilant/futura/watcher/internal/handlers/webhook"
	"github.com/opisvigilant/futura/watcher/internal/models"
)

// Handler is implemented by any handler.
// The Handle method is used to process event
type Handler interface {
	Init(c *config.Configuration) error
	HandleKubernetesEvent(k8sChan <-chan interface{})
	HandleEBpfEvent(ebpfChan <-chan interface{})
	PersistRequest(request *models.Request) error
	PersistTraceEvent(trace *l7_req.TraceEvent) error
}

func New(c *config.Configuration) (Handler, error) {
	var eventHandler Handler
	switch {
	case len(c.Handler.Webhook.URL) > 0:
		eventHandler = new(webhook.Webhook)
	default:
		eventHandler = new(console.Console)
	}
	if err := eventHandler.Init(c); err != nil {
		return nil, err
	}
	return eventHandler, nil
}

// Map maps each event handler function to a name for easily lookup
var Map = map[string]interface{}{
	"console": &console.Console{},
	"webhook": &webhook.Webhook{},
}
