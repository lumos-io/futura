package events

import (
	"context"
	"strings"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/sender"
	"github.com/opisvigilant/futura/watcher/pkg/kubernetes"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/fields"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
)

type KubernetesEventsCollector struct {
	config          *config.Configuration
	stopperChanList []chan struct{}
	startTime       time.Time
	ctx             context.Context
	cancel          context.CancelFunc
}

func New(config *config.Configuration) *KubernetesEventsCollector {
	return &KubernetesEventsCollector{
		startTime: time.Now(),
		config:    config,
	}
}

func (kec *KubernetesEventsCollector) Start(ctx context.Context) error {
	kec.ctx, kec.cancel = context.WithCancel(ctx)

	k8sClient, err := kubernetes.MakeClient(kubernetes.APIConfig{
		AuthType: kubernetes.AuthType(kec.config.Kubernetes.Auth.AuthType),
		Context:  kec.config.Kubernetes.Auth.KubeContextName,
	})
	if err != nil {
		return err
	}

	s, err := sender.New(ctx, kec.config)
	if err != nil {
		return err
	}

	// Detect if events.k8s.io/v1 is available
	discoveryClient := k8sClient.Discovery()
	apiGroups, err := discoveryClient.ServerGroups()
	if err != nil {
		return err
	}

	eventsV1Available := false
	for _, g := range apiGroups.Groups {
		if g.Name == "events.k8s.io" {
			for _, v := range g.Versions {
				if v.Version == "v1" {
					eventsV1Available = true
				}
			}
		}
	}

	log.Info().Msgf("events.k8s.io/v1 available: %v", eventsV1Available)

	if len(kec.config.Kubernetes.Namespaces) == 0 {
		kec.startWatch(corev1.NamespaceAll, k8sClient, s, eventsV1Available)
	} else {
		for _, ns := range kec.config.Kubernetes.Namespaces {
			kec.startWatch(ns, k8sClient, s, eventsV1Available)
		}
	}
	return nil
}

func (kec *KubernetesEventsCollector) Shutdown(ctx context.Context) error {
	if kec.cancel == nil {
		return nil
	}
	for _, stopperChan := range kec.stopperChanList {
		close(stopperChan)
	}
	kec.cancel()
	return nil
}

func (kec *KubernetesEventsCollector) startWatch(ns string, client k8s.Interface, sender *sender.Sender, useEventsV1 bool) {
	stopperChan := make(chan struct{})
	kec.stopperChanList = append(kec.stopperChanList, stopperChan)

	if useEventsV1 {
		kec.startEventsV1Watch(client, ns, sender, stopperChan)
	} else {
		kec.startCoreV1Watch(client, ns, sender, stopperChan)
	}
}

// -------- core/v1 events --------

func (kec *KubernetesEventsCollector) startCoreV1Watch(clientset k8s.Interface, ns string, sender *sender.Sender, stopper chan struct{}) {
	client := clientset.CoreV1().RESTClient()
	watchList := cache.NewListWatchFromClient(client, "events", ns, fields.Everything())

	_, controller := cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher:   watchList,
		ResyncPeriod:    10 * time.Minute,
		ObjectType:      &corev1.Event{},
		MinWatchTimeout: 10 * time.Minute,
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj any) {
				if ev, ok := obj.(*corev1.Event); ok {
					kec.handleCoreV1Event(ev, sender)
				}
			},
			UpdateFunc: func(_, obj any) {
				if ev, ok := obj.(*corev1.Event); ok {
					kec.handleCoreV1Event(ev, sender)
				}
			},
			DeleteFunc: func(obj any) {
				if ev, ok := obj.(*corev1.Event); ok {
					kec.handleCoreV1Event(ev, sender)
				}
			},
		},
	})

	go func() {
		log.Info().Msgf("starting core/v1 events informer for ns=%q", ns)
		controller.Run(stopper)
		log.Info().Msgf("stopped core/v1 events informer for ns=%q", ns)
	}()
}

// -------- events.k8s.io/v1 events --------

func (kec *KubernetesEventsCollector) startEventsV1Watch(clientset k8s.Interface, ns string, sender *sender.Sender, stopper chan struct{}) {
	client := clientset.EventsV1().RESTClient()
	watchList := cache.NewListWatchFromClient(client, "events", ns, fields.Everything())

	_, controller := cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher:   watchList,
		ResyncPeriod:    10 * time.Minute,
		ObjectType:      &eventsv1.Event{},
		MinWatchTimeout: 10 * time.Minute,
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj any) {
				if ev, ok := obj.(*eventsv1.Event); ok {
					kec.handleEventsV1Event(ev, sender)
				}
			},
			UpdateFunc: func(_, obj any) {
				if ev, ok := obj.(*eventsv1.Event); ok {
					kec.handleEventsV1Event(ev, sender)
				}
			},
			DeleteFunc: func(obj any) {
				if ev, ok := obj.(*eventsv1.Event); ok {
					kec.handleEventsV1Event(ev, sender)
				}
			},
		},
	})

	go func() {
		log.Info().Msgf("starting events.k8s.io/v1 informer for ns=%q", ns)
		controller.Run(stopper)
		log.Info().Msgf("stopped events.k8s.io/v1 informer for ns=%q", ns)
	}()
}

// -------- event handlers --------

var severityMap = map[string]int{
	"normal":  9,
	"warning": 13,
}

func (kec *KubernetesEventsCollector) handleCoreV1Event(ev *corev1.Event, sender *sender.Sender) {
	eventTimestamp := getCoreV1EventTimestamp(ev)
	if eventTimestamp.IsZero() {
		log.Debug().Msgf("core/v1 event %s has no timestamp, skipping", ev.Name)
		return
	}

	kev := &pb.KubernetesEvent{
		ObjectKind:            ev.InvolvedObject.Kind,
		ObjectName:            ev.InvolvedObject.Name,
		ObjectUid:             string(ev.InvolvedObject.UID),
		ObjectFieldpath:       ev.InvolvedObject.FieldPath,
		ObjectApiVersion:      ev.InvolvedObject.APIVersion,
		ObjectResourceVersion: ev.InvolvedObject.ResourceVersion,
		ObjectTimestamp:       eventTimestamp.Unix(),
		ObjectNamespace:       ev.InvolvedObject.Namespace,
		EventMessage:          ev.Message,
		EventReason:           ev.Reason,
		EventAction:           ev.Action,
		EventStarttime:        ev.CreationTimestamp.String(),
		EventName:             ev.Name,
		EventUid:              string(ev.UID),
		NodeName:              ev.Source.Host,
	}

	if severityNumber, ok := severityMap[strings.ToLower(ev.Type)]; ok {
		kev.EventSeverityNumber = int64(severityNumber)
		kev.EventSeverityText = ev.Type
	}

	if ev.Count != 0 {
		kev.EventCount = int64(ev.Count)
	}

	// Extract scheduling constraints for failed scheduling events
	if ev.Reason == "FailedScheduling" && ev.InvolvedObject.Kind == "Pod" {
		kev.SchedulingConstraints = extractSchedulingConstraints(ev.Message)
	}

	log.Trace().
		Str("event", ev.Name).
		Str("namespace", ev.Namespace).
		Str("reason", ev.Reason).
		Msg("processed core/v1 Kubernetes event")

	sender.KubernetesEventChan <- kev
}

func (kec *KubernetesEventsCollector) handleEventsV1Event(ev *eventsv1.Event, sender *sender.Sender) {
	eventTimestamp := getEventV1EventTimestamp(ev)
	if eventTimestamp.IsZero() {
		log.Debug().Msgf("events.k8s.io/v1 event %s has no timestamp, skipping", ev.Name)
		return
	}

	kev := &pb.KubernetesEvent{
		ObjectKind:            ev.Regarding.Kind,
		ObjectName:            ev.Regarding.Name,
		ObjectUid:             string(ev.Regarding.UID),
		ObjectFieldpath:       ev.Regarding.FieldPath,
		ObjectApiVersion:      ev.Regarding.APIVersion,
		ObjectResourceVersion: ev.ResourceVersion,
		ObjectTimestamp:       eventTimestamp.Unix(),
		ObjectNamespace:       ev.Regarding.Namespace,
		EventMessage:          ev.Note,
		EventReason:           ev.Reason,
		EventAction:           ev.Action,
		EventStarttime:        ev.CreationTimestamp.String(),
		EventName:             ev.Name,
		EventUid:              string(ev.UID),
		NodeName:              ev.DeprecatedSource.Host,
	}

	if severityNumber, ok := severityMap[strings.ToLower(string(ev.Type))]; ok {
		kev.EventSeverityNumber = int64(severityNumber)
		kev.EventSeverityText = string(ev.Type)
	}

	if ev.DeprecatedCount != 0 {
		kev.EventCount = int64(ev.DeprecatedCount)
	}

	// Extract scheduling constraints for failed scheduling events
	if ev.Reason == "FailedScheduling" && ev.Regarding.Kind == "Pod" {
		kev.SchedulingConstraints = extractSchedulingConstraints(ev.Note)
	}

	log.Trace().
		Str("event", ev.Name).
		Str("namespace", ev.Namespace).
		Str("reason", ev.Reason).
		Msg("processed events.k8s.io/v1 Kubernetes event")

	sender.KubernetesEventChan <- kev
}

// -------- timestamp helpers --------

func getCoreV1EventTimestamp(ev *corev1.Event) time.Time {
	switch {
	case !ev.EventTime.Time.IsZero():
		return ev.EventTime.Time
	case !ev.LastTimestamp.Time.IsZero():
		return ev.LastTimestamp.Time
	case !ev.FirstTimestamp.Time.IsZero():
		return ev.FirstTimestamp.Time
	default:
		return time.Now()
	}
}

func getEventV1EventTimestamp(evt *eventsv1.Event) time.Time {
	switch {
	case !evt.EventTime.IsZero():
		return evt.EventTime.Time
	case !evt.DeprecatedLastTimestamp.IsZero():
		return evt.DeprecatedLastTimestamp.Time
	case !evt.ObjectMeta.CreationTimestamp.IsZero():
		return evt.ObjectMeta.CreationTimestamp.Time
	default:
		return time.Now()
	}
}

// extractSchedulingConstraints parses failed scheduling event messages to extract constraint details
func extractSchedulingConstraints(message string) *pb.SchedulingConstraints {
	if message == "" {
		return nil
	}

	constraints := &pb.SchedulingConstraints{}

	// Try to extract resource requirements if the message mentions specific values
	// Example: "pod requests: cpu=100m, memory=128Mi"
	if strings.Contains(message, "cpu=") {
		if cpu := extractResourceValue(message, "cpu="); cpu != "" {
			constraints.RequiredCpu = cpu
		}
	}
	if strings.Contains(message, "memory=") {
		if memory := extractResourceValue(message, "memory="); memory != "" {
			constraints.RequiredMemory = memory
		}
	}

	// Parse common scheduling failure patterns
	msgLower := strings.ToLower(message)

	// Extract insufficient resource information
	if strings.Contains(msgLower, "insufficient cpu") {
		constraints.InsufficientResource = "cpu"
	} else if strings.Contains(msgLower, "insufficient memory") {
		constraints.InsufficientResource = "memory"
	} else if strings.Contains(msgLower, "too many pods") || strings.Contains(msgLower, "insufficient pods") {
		constraints.InsufficientResource = "pods"
	} else if strings.Contains(msgLower, "insufficient storage") {
		constraints.InsufficientResource = "storage"
	}

	// Extract resource requirements from messages like "0/3 nodes are available: 3 Insufficient cpu."
	// or "0/5 nodes are available: 2 node(s) had taint {node-role.kubernetes.io/master: }, that the pod didn't tolerate"
	if strings.Contains(message, "nodes are available") {
		parts := strings.Split(message, ":")
		if len(parts) >= 2 {
			// Extract the number of nodes considered
			if beforeColon := strings.TrimSpace(parts[0]); strings.Contains(beforeColon, "/") {
				nodeParts := strings.Split(beforeColon, "/")
				if len(nodeParts) >= 2 {
					if considered := extractNumber(nodeParts[1]); considered > 0 {
						constraints.NodesConsidered = considered
					}
				}
			}

			// Parse reasons after the colon
			reasonText := strings.Join(parts[1:], ":")
			parseSchedulingReasons(reasonText, constraints)
		}
	}

	// Extract node selector requirements
	if strings.Contains(msgLower, "node(s) didn't match node selector") ||
	   strings.Contains(msgLower, "node selector") {
		// This is a simplified extraction - in practice, you might want to parse more details
		constraints.NodeSelectorRequirements = []string{"node-selector-constraint"}
	}

	// Extract affinity constraints
	if strings.Contains(msgLower, "node(s) didn't match pod affinity") ||
	   strings.Contains(msgLower, "affinity") {
		constraints.AffinityRequirements = []string{"pod-affinity-constraint"}
	}

	// Extract anti-affinity constraints
	if strings.Contains(msgLower, "node(s) didn't match pod anti-affinity") ||
	   strings.Contains(msgLower, "anti-affinity") {
		constraints.AntiAffinityConflicts = []string{"pod-anti-affinity-constraint"}
	}

	// Extract taint/toleration issues
	if strings.Contains(msgLower, "taint") && strings.Contains(msgLower, "tolerate") {
		constraints.UnsatisfiedTolerations = []string{"taint-toleration-constraint"}
	}

	return constraints
}

// parseSchedulingReasons parses the reason part of scheduling failure messages
func parseSchedulingReasons(reasonText string, constraints *pb.SchedulingConstraints) {
	reasonLower := strings.ToLower(reasonText)

	// Count different types of node rejections
	if strings.Contains(reasonLower, "insufficient cpu") || strings.Contains(reasonLower, "insufficient memory") {
		if num := extractNodeCount(reasonText, "insufficient"); num > 0 {
			constraints.NodesRejectedResource = num
		}
	}

	if strings.Contains(reasonLower, "taint") {
		if num := extractNodeCount(reasonText, "taint"); num > 0 {
			constraints.NodesRejectedTaints = num
		}
	}

	if strings.Contains(reasonLower, "affinity") || strings.Contains(reasonLower, "anti-affinity") {
		if num := extractNodeCount(reasonText, "affinity"); num > 0 {
			constraints.NodesRejectedAffinity = num
		}
	}

	if strings.Contains(reasonLower, "node selector") {
		if num := extractNodeCount(reasonText, "node selector"); num > 0 {
			constraints.NodesRejectedSelector = num
		}
	}
}

// extractNumber extracts the first number found in a string
func extractNumber(s string) int32 {
	var num int32
	for _, char := range s {
		if char >= '0' && char <= '9' {
			num = num*10 + int32(char-'0')
		} else if num > 0 {
			break
		}
	}
	return num
}

// extractNodeCount extracts node count from reason text for specific constraint types
func extractNodeCount(text string, constraintType string) int32 {
	// Look for patterns like "3 node(s) had taint" or "2 Insufficient cpu"
	words := strings.Fields(text)
	for i, word := range words {
		if num := extractNumber(word); num > 0 {
			// Check if this number is related to our constraint type
			remaining := strings.Join(words[i:], " ")
			if strings.Contains(strings.ToLower(remaining), constraintType) {
				return num
			}
		}
	}
	return 0
}

// extractResourceValue extracts resource values like "100m" from "cpu=100m" patterns
func extractResourceValue(text, prefix string) string {
	if idx := strings.Index(text, prefix); idx != -1 {
		start := idx + len(prefix)
		end := start
		for end < len(text) && (text[end] != ',' && text[end] != ' ' && text[end] != ')' && text[end] != ']') {
			end++
		}
		if end > start {
			return text[start:end]
		}
	}
	return ""
}
