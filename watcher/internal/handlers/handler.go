package handlers

import (
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/event"
	"github.com/opisvigilant/futura/watcher/internal/handlers/console"
	"github.com/opisvigilant/futura/watcher/internal/handlers/webhook"
)

// Handler is implemented by any handler.
// The Handle method is used to process event
type Handler interface {
	Init(c *config.Configuration) error
	Handle(e event.Event)
}

// Map maps each event handler function to a name for easily lookup
var Map = map[string]interface{}{
	"console": &console.Console{},
	"webhook": &webhook.Webhook{},
}
