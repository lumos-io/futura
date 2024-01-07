package console

import (
	"os"

	"github.com/opisvigilant/futura/pkg/logger"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/event"
	"github.com/rs/zerolog"
)

// Console handler implements Handler interface,
// print each event with JSON format
type Console struct {
}

// Init initializes handler configuration
// Do nothing for default handler
func (d *Console) Init(c *config.Configuration) error {
	if c.Handler.Console.Color {
		logger.Logger().Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	return nil
}

// Handle handles an event.
func (d *Console) Handle(e event.Event) {
	logger.Logger().Info().Msg(e.Message())
}
