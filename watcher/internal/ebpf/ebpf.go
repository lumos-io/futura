package ebpf

type EbpfCollector struct {
}

func NewEbpfCollector() *EbpfCollector {
	c := &EbpfCollector{}
	return c
}

func (e *EbpfCollector) Start() error {
	return nil
}
