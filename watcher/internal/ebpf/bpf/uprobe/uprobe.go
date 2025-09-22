package uprobe

import (
	"context"
	"log"
	"time"

	"github.com/cilium/ebpf/rlimit"
)

type Uprobes struct {
	Objects *uprobeObjects
}

func NewUprobes() *Uprobes {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}

	objs := &uprobeObjects{}
	if err := loadUprobeObjects(objs, nil); err != nil {
		log.Fatalf("loading uprobe objects: %v", err)
	}

	return &Uprobes{
		Objects: objs,
	}
}

// Poll implements CollectorProgram and listens until ctx is canceled.
func (u *Uprobes) Poll(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			var value uint64
			if err := u.Objects.RpsCount.Lookup(uint32(0), &value); err != nil {
				log.Printf("reading map: %v", err)
				continue
			}
			log.Printf("Uprobe called %d times\n", value)
		}
	}
}

func (u *Uprobes) Close() error {
	u.Objects.Close()
	return u.Objects.Close()
}
