package simulator

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"

	pbcl "github.com/opisvigilant/futura/proto/gen/cluster"
	pbev "github.com/opisvigilant/futura/proto/gen/events"
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
)

func StreamKubeletEvents(ctx context.Context, wg *sync.WaitGroup, nodeName string, pods []*pbst.PodStats, eventsChn chan<- *pbev.KubernetesEvent, clusterObjCh chan<- *pbcl.KubernetesClusterObject) {
	defer wg.Done()

	// Keep a copy of last seen pods to detect changes
	lastPods := make(map[string]*pbst.PodStats)
	for _, p := range pods {
		lastPods[p.PodRef.Uid] = p
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Simulate pod churn
			// (In reality you’d share the same churn logic as streamKubeletStats)
			if rand.Intn(10) == 0 && len(pods) > 3 {
				idx := rand.Intn(len(pods))
				removed := pods[idx]
				pods = append(pods[:idx], pods[idx+1:]...)
				delete(lastPods, removed.PodRef.Uid)

				eventsChn <- &pbev.KubernetesEvent{
					ObjectKind:        "Pod",
					ObjectName:        removed.PodRef.Name,
					ObjectNamespace:   removed.PodRef.Namespace,
					ObjectUid:         removed.PodRef.Uid,
					ObjectTimestamp:   time.Now().Unix(),
					EventSeverityText: "Normal",
					EventReason:       "Killing",
					EventAction:       "PodDeleted",
					EventStarttime:    time.Now().Format(time.RFC3339),
					EventName:         fmt.Sprintf("delete-%s", removed.PodRef.Name),
					EventMessage:      fmt.Sprintf("Deleted pod: %s", removed.PodRef.Name),
					EventUid:          uuid.NewString(),
					EventCount:        1,
					NodeName:          nodeName,
					ObjectApiVersion:  "v1",
					Enrichment: &pbev.EnrichmentMetadata{
						OrganizationId: 42,
						ClusterId:      12345,
						ReceivedAtUnix: time.Now().Unix(),
					},
				}

				clusterObjCh <- GenerateClusterObjectFromPod(nodeName, removed, "delete")
			}

			if rand.Intn(10) == 0 {
				newPod := newRandomPod()
				pods = append(pods, newPod)
				lastPods[newPod.PodRef.Uid] = newPod

				eventsChn <- &pbev.KubernetesEvent{
					ObjectKind:        "Pod",
					ObjectName:        newPod.PodRef.Name,
					ObjectNamespace:   newPod.PodRef.Namespace,
					ObjectUid:         newPod.PodRef.Uid,
					ObjectTimestamp:   time.Now().Unix(),
					EventSeverityText: "Normal",
					EventReason:       "Scheduled",
					EventAction:       "PodCreated",
					EventStarttime:    time.Now().Format(time.RFC3339),
					EventName:         fmt.Sprintf("create-%s", newPod.PodRef.Name),
					EventMessage:      fmt.Sprintf("Created pod: %s", newPod.PodRef.Name),
					EventUid:          uuid.NewString(),
					EventCount:        1,
					NodeName:          nodeName,
					ObjectApiVersion:  "v1",
					Enrichment: &pbev.EnrichmentMetadata{
						OrganizationId: 42,
						ClusterId:      12345,
						ReceivedAtUnix: time.Now().Unix(),
					},
				}
				clusterObjCh <- GenerateClusterObjectFromPod(nodeName, newPod, "snapshot")
			}

			// Random status change event for an existing pod
			if len(pods) > 0 && rand.Intn(5) == 0 {
				target := pods[rand.Intn(len(pods))]
				eventsChn <- &pbev.KubernetesEvent{
					ObjectKind:        "Pod",
					ObjectName:        target.PodRef.Name,
					ObjectNamespace:   target.PodRef.Namespace,
					ObjectUid:         target.PodRef.Uid,
					ObjectTimestamp:   time.Now().Unix(),
					EventSeverityText: "Warning",
					EventReason:       "BackOff",
					EventAction:       "CrashLoopBackOff",
					EventStarttime:    time.Now().Format(time.RFC3339),
					EventName:         fmt.Sprintf("backoff-%s", target.PodRef.Name),
					EventMessage:      fmt.Sprintf("Pod %s is in CrashLoopBackOff", target.PodRef.Name),
					EventUid:          uuid.NewString(),
					EventCount:        int64(rand.Intn(5) + 1),
					NodeName:          nodeName,
					ObjectApiVersion:  "v1",
					Enrichment: &pbev.EnrichmentMetadata{
						OrganizationId: 42,
						ClusterId:      12345,
						ReceivedAtUnix: time.Now().Unix(),
					},
				}
				clusterObjCh <- GenerateClusterObjectFromPod(nodeName, target, "update")
			}

			time.Sleep(50 * time.Millisecond) // match with stats loop pace
		}
	}
}

func randomChoice[T any](arr []T) T {
	return arr[rand.Intn(len(arr))]
}
