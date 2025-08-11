package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"

	pbev "github.com/opisvigilant/futura/proto/gen/events"
)

// eventGenerator sends N events into the channel and then closes it
func eventGenerator(out chan<- *pbev.KubernetesEvent, numEvents int) {
	defer close(out)
	for range numEvents {
		out <- generateRandomEvent()
		// Optional delay to simulate streaming
		time.Sleep(50 * time.Millisecond)
	}
}

func generateRandomEvent() *pbev.KubernetesEvent {
	now := time.Now().Unix()

	objectKinds := []string{"Pod", "Node", "Deployment", "Service", "ReplicaSet", "ConfigMap"}
	namespaces := []string{"default", "kube-system", "prod", "staging", "dev"}
	nodes := []string{"node-a", "node-b", "node-c"}
	severityTexts := []string{"Normal", "Warning", "Error"}
	reasons := []string{"Created", "Scheduled", "Failed", "BackOff", "Pulled", "Killing"}
	actions := []string{"StartContainer", "StopContainer", "DeletePod", "ScaleUp", "ScaleDown"}
	k8sVersions := []string{"1.26.0", "1.27.3", "1.28.1"}

	return &pbev.KubernetesEvent{
		ObjectKind:            randomChoice(objectKinds),
		ObjectName:            fmt.Sprintf("%s-%d", randomString(5), rand.Intn(100)),
		ObjectUid:             uuid.NewString(),
		ObjectFieldpath:       "",
		ObjectTimestamp:       now - int64(rand.Intn(3600)),
		ObjectNamespace:       randomChoice(namespaces),
		EventSeverityNumber:   int64(rand.Intn(3)),
		EventSeverityText:     randomChoice(severityTexts),
		EventReason:           randomChoice(reasons),
		EventAction:           randomChoice(actions),
		EventStarttime:        time.Now().Add(-time.Duration(rand.Intn(300)) * time.Second).Format(time.RFC3339),
		EventName:             fmt.Sprintf("event-%s", uuid.NewString()),
		EventMessage:          fmt.Sprintf("%s %s successfully", randomChoice(actions), randomChoice(objectKinds)),
		EventUid:              uuid.NewString(),
		EventCount:            int64(rand.Intn(5) + 1),
		ObjectApiVersion:      "v1",
		ObjectResourceVersion: fmt.Sprintf("%d", rand.Intn(100000)),
		NodeName:              randomChoice(nodes),
		Enrichment: &pbev.EnrichmentMetadata{
			OrganizationId: uint32(rand.Intn(1000) + 1),
			ClusterId:      int64(rand.Intn(1000000)),
			K8SVersion:     randomChoice(k8sVersions),
			ReceivedAtUnix: now,
		},
	}
}

func randomChoice[T any](arr []T) T {
	return arr[rand.Intn(len(arr))]
}

func randomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz")
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
