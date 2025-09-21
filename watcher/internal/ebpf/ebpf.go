package ebpf

import (
	"context"
	"sync"

	"github.com/opisvigilant/futura/watcher/internal/ebpf/bpf/uprobe"
	"github.com/rs/zerolog/log"
)

// CollectorProgram defines a common interface for eBPF programs.
type CollectorProgram interface {
	// Poll runs the program until the context is canceled.
	Poll(ctx context.Context) error
	// Close releases resources.
	Close() error
}

type EbpfCollector struct {
	programs []CollectorProgram
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewEbpfCollector() *EbpfCollector {
	programs := []CollectorProgram{
		uprobe.NewUprobes(),
	}

	c := &EbpfCollector{
		programs: programs,
	}
	return c
}

func (e *EbpfCollector) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	for _, prog := range e.programs {
		e.wg.Add(1)
		go func(p CollectorProgram) {
			defer e.wg.Done()
			if err := p.Poll(ctx); err != nil {
				log.Error().Err(err).Msg("program poll error")
			}
		}(prog)
	}
	return nil
}

// Stop all programs gracefully
func (e *EbpfCollector) Close() error {
	if e.cancel != nil {
		e.cancel()
	}

	e.wg.Wait()

	for _, prog := range e.programs {
		prog.Close()
	}
	return nil
}
