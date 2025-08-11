package main

import (
	"encoding/json"
	"fmt"

	pbev "github.com/opisvigilant/futura/proto/gen/events"
	// pbcl "github.com/opisvigilant/futura/proto/gen/cluster"
	// pbcm "github.com/opisvigilant/futura/proto/gen/common"
	// pbst "github.com/opisvigilant/futura/proto/gen/stats"
)

func main() {
	eventCh := make(chan *pbev.KubernetesEvent)

	// Start the generator goroutine
	go eventGenerator(eventCh, 50)

	// You could start more goroutines for other work here
	// go myOtherRoutine()

	// Consume events from the channel
	for ev := range eventCh {
		b, _ := json.Marshal(ev)
		fmt.Println(string(b))
	}
}
