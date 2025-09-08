package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	"github.com/opisvigilant/futura/simulator/simulator"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wg := &sync.WaitGroup{}

	numNodes := 10
	metricsCh := make(chan *pb.KubernetesKubeletStats, 1000)
	eventsCh := make(chan *pb.KubernetesEvent, 1000)
	clusterObjCh := make(chan *pb.KubernetesClusterObject, 1000)

	// Launch one goroutine per node
	for i := 1; i <= numNodes; i++ {
		wg.Add(2)
		node := fmt.Sprintf("node-%03d", i)
		// Keep a small set of "current" pods to simulate churn
		pods := simulator.RandomPods()

		go simulator.StreamKubeletStats(ctx, wg, node, pods, metricsCh)
		go simulator.StreamKubeletEvents(ctx, wg, node, pods, eventsCh, clusterObjCh)
	}

	// Start consumers (file writers)
	wg.Add(3)
	go consumeToFile(ctx, wg, metricsCh, "./output/metrics.jsonl")
	go consumeToFile(ctx, wg, eventsCh, "./output/events.jsonl")
	go consumeToFile(ctx, wg, clusterObjCh, "./output/cluster_objects.jsonl")

	// Wait for context timeout
	<-ctx.Done()

	close(metricsCh)
	close(eventsCh)
	close(clusterObjCh)

	wg.Wait()
}

// Generic consumer to file
func consumeToFile[T any](ctx context.Context, wg *sync.WaitGroup, ch <-chan T, filename string) {
	defer wg.Done()

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filename, err)
	}
	defer file.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-ch:
			if !ok {
				return
			}
			b, err := json.Marshal(m)
			if err == nil {
				file.Write(append(b, '\n'))
			}
		}
	}
}
