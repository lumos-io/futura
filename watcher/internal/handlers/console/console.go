package console

import (
	"os"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/ebpf/l7_req"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/internal/models"
	"github.com/rs/zerolog"
)

// Console handler implements Handler interface,
// print each event with JSON format
type Console struct {
}

// Init initializes handler configuration
// Do nothing for default handler
func (csl *Console) Init(c *config.Configuration) error {
	if c.Handler.Console.Color {
		logger.Logger().Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	return nil
}

// Handle handles an event.
func (csl *Console) HandleKubernetesEvent(k8sChan <-chan interface{}) {
}

func (csl *Console) HandleEBpfEvent(ebpfChan <-chan interface{}) {
}

func (csl *Console) PersistRequest(request *models.Request) error {
	return nil
}

func (csl *Console) PersistTraceEvent(trace *l7_req.TraceEvent) error {
	return nil
}
