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
	log.Logger.Info().Msg("starting to watch namespaces for the events.")
	if len(kec.config.Kubernetes.Namespaces) == 0 {
		kec.startWatch(corev1.NamespaceAll, k8sClient, s)
	} else {
		for _, ns := range kec.config.Kubernetes.Namespaces {
			kec.startWatch(ns, k8sClient, s)
		}
	}
	return nil
}

func (kec *KubernetesEventsCollector) Shutdown(ctx context.Context) error {
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

func (kec *KubernetesEventsCollector) startWatch(ns string, client k8s.Interface, sender *sender.Sender) {
	stopperChan := make(chan struct{})
	kec.stopperChanList = append(kec.stopperChanList, stopperChan)
	kec.startWatchingNamespace(client, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if ev, ok := obj.(*corev1.Event); ok {
				kec.handleEvent(ev, sender)
			} else {
				log.Warn().Msg("received non-Event object from informer")
			}
		},
		UpdateFunc: func(_, obj any) {
			if ev, ok := obj.(*corev1.Event); ok {
				kec.handleEvent(ev, sender)
			} else {
				log.Warn().Msg("received non-Event object from informer")
			}
		},
		DeleteFunc: func(obj any) {
			if ev, ok := obj.(*corev1.Event); ok {
				kec.handleEvent(ev, sender)
			} else {
				log.Warn().Msg("received non-Event object from informer")
			}
		},
	}, ns, stopperChan)
}

// startWatchingNamespace creates an informer and starts
// watching a specific namespace for the events.
func (kec *KubernetesEventsCollector) startWatchingNamespace(clientset k8s.Interface, handlers cache.ResourceEventHandlerFuncs, ns string, stopper chan struct{}) {
	client := clientset.CoreV1().RESTClient()
	watchList := cache.NewListWatchFromClient(client, "events", ns, fields.Everything())
	_, controller := cache.NewInformerWithOptions(cache.InformerOptions{
		ListerWatcher: watchList,
		ObjectType:    &corev1.Event{},
		// ResyncPeriod:  0,
		Handler: handlers,
	})
	go controller.Run(stopper)
}

var severityMap = map[string]int{
	"normal":  9,
	"warning": 13,
}

func (kec *KubernetesEventsCollector) handleEvent(ev *corev1.Event, sender *sender.Sender) {
	eventTimestamp := getEventTimestamp(ev)
	if eventTimestamp.IsZero() {
		log.Debug().Msgf("event %s has no timestamp, skipping", ev.Name)
	}
	// extract event
	kev := &pb.KubernetesEvent{
		ObjectKind:            ev.InvolvedObject.Kind,
		ObjectName:            ev.InvolvedObject.Name,
		ObjectUid:             string(ev.InvolvedObject.UID),
		ObjectFieldpath:       ev.InvolvedObject.FieldPath,
		ObjectApiVersion:      ev.InvolvedObject.APIVersion,
		ObjectResourceVersion: ev.InvolvedObject.ResourceVersion,
		ObjectTimestamp:       eventTimestamp.UnixMilli(),
		ObjectNamespace:       ev.InvolvedObject.Namespace,
		EventMessage:          ev.Message,
		EventReason:           ev.Reason,
		EventAction:           ev.Action,
		EventStarttime:        ev.CreationTimestamp.String(),
		EventName:             ev.Name,
		EventUid:              string(ev.UID),
		NodeName:              ev.Source.Host,
	}

	// Set the "SeverityNumber" and "SeverityText" if a known type of
	// severity is found.
	if severityNumber, ok := severityMap[strings.ToLower(ev.Type)]; ok {
		kev.EventSeverityNumber = int64(severityNumber)
		kev.EventSeverityText = ev.Type
	} else {
		log.Logger.Debug().Msgf("unknown severity type %s", ev.Type)
	}

	// "Count" field of k8s event will be '0' in case it is
	// not present in the collected event from k8s.
	if ev.Count != 0 {
		kev.EventCount = int64(ev.Count)
	}

	log.Logger.Trace().
		Str("event", ev.Name).
		Str("namespace", ev.Namespace).
		Str("reason", ev.Reason).
		Msg("processed Kubernetes event")

	// send it to a channel for the sender
	sender.KubernetesEventChan <- kev
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
