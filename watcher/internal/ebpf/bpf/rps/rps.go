package rps

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cilium/ebpf/rlimit"
)

type RPS struct {
	Objects *rpsObjects
}

func NewRPS() *RPS {
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal(err)
	}

	objs := &rpsObjects{}
	if err := loadRpsObjects(objs, nil); err != nil {
		log.Fatalf("loading uprobe objects: %v", err)
	}

	return &RPS{
		Objects: objs,
	}
}

// Poll implements CollectorProgram and listens until ctx is canceled.
func (u *RPS) Poll(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			iter := u.Objects.Rps.Iterate()
			var cgid, count uint64

			for iter.Next(&cgid, &count) {
				fmt.Printf("cgroup=%d RPS=%d\n", cgid, count)
				// reset counter
				zero := uint64(0)
				u.Objects.Rps.Put(cgid, zero)
			}
			if err := iter.Err(); err != nil {
				log.Printf("iteration error: %v", err)
			}
		}
	}
}

func (u *RPS) Close() error {
	u.Objects.Close()
	return u.Objects.Close()
}
