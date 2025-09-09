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
