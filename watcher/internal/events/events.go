package events

import (
	"context"
	"strings"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	k8s "github.com/opisvigilant/futura/watcher/pkg/kubernetes"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
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

	k8sClient, err := k8s.New(kec.config.Kubernetes.InCluster)
	if err != nil {
		return err
	}

	logger.Logger().Info().Msg("starting to watch namespaces for the events.")
	if len(kec.config.Kubernetes.Namespaces) == 0 {
		kec.startWatch(corev1.NamespaceAll, k8sClient)
	} else {
		for _, ns := range kec.config.Kubernetes.Namespaces {
			kec.startWatch(ns, k8sClient)
		}
	}
	return nil
}

func (kec *KubernetesEventsCollector) Shutdown(context.Context) error {
	if kec.cancel == nil {
		return nil
	}
	// Stop watching all the namespaces by closing all the stopper channels.
	for _, stopperChan := range kec.stopperChanList {
		close(stopperChan)
	}
	kec.cancel()
	return nil
}

func (kec *KubernetesEventsCollector) startWatch(ns string, client *k8s.Client) {
	stopperChan := make(chan struct{})
	kec.stopperChanList = append(kec.stopperChanList, stopperChan)
	kec.startWatchingNamespace(client, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			ev := obj.(*corev1.Event)
			kec.handleEvent(ev)
		},
		UpdateFunc: func(_, obj any) {
			ev := obj.(*corev1.Event)
			kec.handleEvent(ev)
		},
	}, ns, stopperChan)
}

// startWatchingNamespace creates an informer and starts
// watching a specific namespace for the events.
func (kec *KubernetesEventsCollector) startWatchingNamespace(clientset *k8s.Client, handlers cache.ResourceEventHandlerFuncs, ns string, stopper chan struct{}) {
	client := clientset.RawClient().CoreV1().RESTClient()
	watchList := cache.NewListWatchFromClient(client, "events", ns, fields.Everything())
	_, controller := cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher: watchList,
		ObjectType:    &corev1.Event{},
		ResyncPeriod:  0,
		Handler:       handlers,
	})
	go controller.Run(stopper)
}

func (kec *KubernetesEventsCollector) handleEvent(ev *corev1.Event) {
	if kec.allowEvent(ev) {
		// extract event

		// send it to a channel for the sender
	}
}

// Allow events with eventTimestamp(EventTime/LastTimestamp/FirstTimestamp)
// not older than the receiver start time so that
// event flood can be avoided upon startup.
func (kec *KubernetesEventsCollector) allowEvent(ev *corev1.Event) bool {
	eventTimestamp := getEventTimestamp(ev)
	return !eventTimestamp.Before(kec.startTime)
}

// Return the EventTimestamp based on the populated k8s event timestamps.
// Priority: EventTime > LastTimestamp > FirstTimestamp.
func getEventTimestamp(ev *corev1.Event) time.Time {
	var eventTimestamp time.Time

	switch {
	case ev.EventTime.Time != time.Time{}:
		eventTimestamp = ev.EventTime.Time
	case ev.LastTimestamp.Time != time.Time{}:
		eventTimestamp = ev.LastTimestamp.Time
	case ev.FirstTimestamp.Time != time.Time{}:
		eventTimestamp = ev.FirstTimestamp.Time
	}

	return eventTimestamp
}

var severityMap = map[string]int{
	"normal":  9,
	"warning": 13,
}

type KubernetesEvent struct {
	ObjectKind          string
	ObjectName          string
	ObjectUID           string
	ObjectFieldPath     string
	ObjectTimestamp     time.Time
	ObjectNamespace     string
	EventSeverityNumber int
	EventSeverityText   string
	EventReason         string
	EventAction         string
	EventStartTime      string
	EventName           string
	EventMessage        string
	EventUID            string
	EventCount          int64
}

func X(ev *corev1.Event) {

	resourceAttrs := rl.Resource().Attributes()
	resourceAttrs.EnsureCapacity(totalResourceAttributes)

	resourceAttrs.PutStr(string(semconv.K8SNodeNameKey), ev.Source.Host)

	// Attributes related to the object causing the event.
	resourceAttrs.PutStr("k8s.object.kind", ev.InvolvedObject.Kind)
	resourceAttrs.PutStr("k8s.object.name", ev.InvolvedObject.Name)
	resourceAttrs.PutStr("k8s.object.uid", string(ev.InvolvedObject.UID))
	resourceAttrs.PutStr("k8s.object.fieldpath", ev.InvolvedObject.FieldPath)
	resourceAttrs.PutStr("k8s.object.api_version", ev.InvolvedObject.APIVersion)
	resourceAttrs.PutStr("k8s.object.resource_version", ev.InvolvedObject.ResourceVersion)

	timestamp := getEventTimestamp(ev)

	// The Message field contains description about the event,
	// which is best suited for the "Body" of the LogRecordSlice.
	message := ev.Message

	// Set the "SeverityNumber" and "SeverityText" if a known type of
	// severity is found.
	if severityNumber, ok := severityMap[strings.ToLower(ev.Type)]; ok {
		sn := severityNumber
		severityText := ev.Type
	} else {
		logger.Debug("unknown severity type", zap.String("type", ev.Type))
	}

	attrs.PutStr("k8s.event.reason", ev.Reason)
	attrs.PutStr("k8s.event.action", ev.Action)
	attrs.PutStr("k8s.event.start_time", ev.CreationTimestamp.String())
	attrs.PutStr("k8s.event.name", ev.Name)
	attrs.PutStr("k8s.event.uid", string(ev.UID))
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), ev.InvolvedObject.Namespace)

	// "Count" field of k8s event will be '0' in case it is
	// not present in the collected event from k8s.
	if ev.Count != 0 {
		attrs.PutInt("k8s.event.count", int64(ev.Count))
	}

}
