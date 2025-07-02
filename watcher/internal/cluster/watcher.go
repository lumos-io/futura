package cluster

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/cluster/cronjob"
	"github.com/opisvigilant/futura/watcher/internal/cluster/daemonset"
	"github.com/opisvigilant/futura/watcher/internal/cluster/deployment"
	"github.com/opisvigilant/futura/watcher/internal/cluster/gvk"
	"github.com/opisvigilant/futura/watcher/internal/cluster/hpa"
	"github.com/opisvigilant/futura/watcher/internal/cluster/jobs"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"github.com/opisvigilant/futura/watcher/internal/cluster/namespace"
	"github.com/opisvigilant/futura/watcher/internal/cluster/node"
	"github.com/opisvigilant/futura/watcher/internal/cluster/pod"
	"github.com/opisvigilant/futura/watcher/internal/cluster/replicaset"
	"github.com/opisvigilant/futura/watcher/internal/cluster/replicationcontroller"
	"github.com/opisvigilant/futura/watcher/internal/cluster/service"
	"github.com/opisvigilant/futura/watcher/internal/cluster/statefulset"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/rs/zerolog/log"

	k8s "github.com/opisvigilant/futura/watcher/pkg/kubernetes"

	quotaclientset "github.com/openshift/client-go/quota/clientset/versioned"
	quotainformersv1 "github.com/openshift/client-go/quota/informers/externalversions"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	// supported distributions
	distributionKubernetes = "kubernetes"
	distributionOpenShift  = "openshift"

	defaultInitialSyncTimeout = 10 * time.Minute
)

type sharedInformer interface {
	Start(<-chan struct{})
	WaitForCacheSync(<-chan struct{}) map[reflect.Type]bool
}

type resourceWatcher struct {
	client            kubernetes.Interface
	osQuotaClient     quotaclientset.Interface
	informerFactories []sharedInformer
	metadataStore     *metadata.Store
	events            chan *metadata.KubernetesResourceEvent

	initialTimeout      time.Duration
	initialSyncDone     *atomic.Bool
	initialSyncTimedOut *atomic.Bool
	config              *config.Configuration
}

// newResourceWatcher creates a Kubernetes resource watcher.
func newResourceWatcher(cfg *config.Configuration, metadataStore *metadata.Store) *resourceWatcher {
	return &resourceWatcher{
		metadataStore:       metadataStore,
		events:              make(chan *metadata.KubernetesResourceEvent),
		initialSyncDone:     &atomic.Bool{},
		initialSyncTimedOut: &atomic.Bool{},
		initialTimeout:      defaultInitialSyncTimeout,
		config:              cfg,
	}
}

func (rw *resourceWatcher) emitEvent(obj any, eventType metadata.EventType) {
	metas := rw.objMetadata(obj)
	for _, m := range metas {
		event := &metadata.KubernetesResourceEvent{
			Type:      eventType,
			Resource:  m.EntityType,
			UID:       string(m.ResourceID),
			Name:      m.Metadata["k8s.name"],
			Namespace: m.Metadata["k8s.namespace.name"],
			Metadata: map[string]string{
				"container.name":  m.Metadata["k8s.container.name"],
				"container.image": m.Metadata["k8s.container.image.name"],
			},
			Timestamp: time.Now(),
		}
		select {
		case rw.events <- event:
			
		default:
			log.Logger.Warn().Msg("Dropping ResourceEvent due to full channel")
		}
	}
}

func (rw *resourceWatcher) initialize() error {
	client, err := k8s.MakeClient(k8s.APIConfig{
		AuthType: k8s.AuthType(rw.config.Kubernetes.AuthType),
		Context:  rw.config.Kubernetes.KubeContextName,
	})
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	rw.client = client

	if rw.config.Kubernetes.Distribution == distributionOpenShift {
		rw.osQuotaClient, err = k8s.MakeOpenShiftQuotaClient(k8s.APIConfig{
			AuthType: k8s.AuthType(rw.config.Kubernetes.AuthType),
			Context:  rw.config.Kubernetes.KubeContextName,
		})
		if err != nil {
			return fmt.Errorf("failed to create OpenShift quota API client: %w", err)
		}
	}

	err = rw.prepareSharedInformerFactory()
	if err != nil {
		return err
	}

	return nil
}

func (rw *resourceWatcher) prepareSharedInformerFactory() error {
	factory := rw.getInformerFactory()

	// Map of supported group version kinds by name of a kind.
	// If none of the group versions are supported by k8s server for a specific kind,
	// informer for that kind won't be set and a warning message is thrown.
	// This map should be kept in sync with what can be provided by the supported k8s server versions.
	supportedKinds := map[string][]schema.GroupVersionKind{
		"Pod":                     {gvk.Pod},
		"Node":                    {gvk.Node},
		"Namespace":               {gvk.Namespace},
		"ReplicationController":   {gvk.ReplicationController},
		"ResourceQuota":           {gvk.ResourceQuota},
		"Service":                 {gvk.Service},
		"DaemonSet":               {gvk.DaemonSet},
		"Deployment":              {gvk.Deployment},
		"ReplicaSet":              {gvk.ReplicaSet},
		"StatefulSet":             {gvk.StatefulSet},
		"Job":                     {gvk.Job},
		"CronJob":                 {gvk.CronJob},
		"HorizontalPodAutoscaler": {gvk.HorizontalPodAutoscaler},
	}

	for kind, gvks := range supportedKinds {
		anySupported := false
		for _, gvk := range gvks {
			supported, err := rw.isKindSupported(gvk)
			if err != nil {
				return err
			}
			if supported {
				anySupported = true
				rw.setupInformerForKind(gvk, factory)
			}
		}
		if !anySupported {
			log.Logger.Warn().Msgf("Server doesn't support any of the group versions defined for the kind `%s`", kind)
		}
	}

	if rw.osQuotaClient != nil {
		quotaFactory := quotainformersv1.NewSharedInformerFactory(rw.osQuotaClient, 0)
		rw.setupInformer(gvk.ClusterResourceQuota, quotaFactory.Quota().V1().ClusterResourceQuotas().Informer())
		rw.informerFactories = append(rw.informerFactories, quotaFactory)
	}
	rw.informerFactories = append(rw.informerFactories, factory)

	return nil
}

func (rw *resourceWatcher) getInformerFactory() informers.SharedInformerFactory {
	factory := informers.NewSharedInformerFactoryWithOptions(
		rw.client,
		rw.config.Kubernetes.MetadataCollectionInterval,
	)
	return factory
}

func (rw *resourceWatcher) isKindSupported(gvk schema.GroupVersionKind) (bool, error) {
	resources, err := rw.client.Discovery().ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		if apierrors.IsNotFound(err) { // if the discovery endpoint isn't present, assume group version is not supported
			log.Logger.Debug().Msg("Group version is not supported")
			return false, nil
		}
		return false, fmt.Errorf("failed to fetch group version details: %w", err)
	}

	for _, r := range resources.APIResources {
		if r.Kind == gvk.Kind {
			return true, nil
		}
	}
	return false, nil
}

func (rw *resourceWatcher) setupInformerForKind(kind schema.GroupVersionKind, factory informers.SharedInformerFactory) {
	switch kind {
	case gvk.Pod:
		rw.setupInformer(kind, factory.Core().V1().Pods().Informer())
	case gvk.Node:
		rw.setupInformer(kind, factory.Core().V1().Nodes().Informer())
	case gvk.Namespace:
		rw.setupInformer(kind, factory.Core().V1().Namespaces().Informer())
	case gvk.ReplicationController:
		rw.setupInformer(kind, factory.Core().V1().ReplicationControllers().Informer())
	case gvk.ResourceQuota:
		rw.setupInformer(kind, factory.Core().V1().ResourceQuotas().Informer())
	case gvk.Service:
		rw.setupInformer(kind, factory.Core().V1().Services().Informer())
	case gvk.DaemonSet:
		rw.setupInformer(kind, factory.Apps().V1().DaemonSets().Informer())
	case gvk.Deployment:
		rw.setupInformer(kind, factory.Apps().V1().Deployments().Informer())
	case gvk.ReplicaSet:
		rw.setupInformer(kind, factory.Apps().V1().ReplicaSets().Informer())
	case gvk.StatefulSet:
		rw.setupInformer(kind, factory.Apps().V1().StatefulSets().Informer())
	case gvk.Job:
		rw.setupInformer(kind, factory.Batch().V1().Jobs().Informer())
	case gvk.CronJob:
		rw.setupInformer(kind, factory.Batch().V1().CronJobs().Informer())
	case gvk.HorizontalPodAutoscaler:
		rw.setupInformer(kind, factory.Autoscaling().V2().HorizontalPodAutoscalers().Informer())
	default:
		log.Logger.Error().Msgf("Could not setup an informer for provided group version kind `%s`", kind.String())
	}
}

// startWatchingResources starts up all informers.
func (rw *resourceWatcher) startWatchingResources(ctx context.Context, inf sharedInformer) context.Context {
	var cancel context.CancelFunc
	timedContextForInitialSync, cancel := context.WithTimeout(ctx, rw.initialTimeout)

	// Start off individual informers in the factory.
	inf.Start(ctx.Done())

	// Ensure cache is synced with initial state, once informers are started up.
	// Note that the event handler can start receiving events as soon as the informers
	// are started. So it's required to ensure that the receiver does not start
	// collecting data before the cache sync since all data may not be available.
	// This method will block either till the timeout set on the context, until
	// the initial sync is complete or the parent context is cancelled.
	inf.WaitForCacheSync(timedContextForInitialSync.Done())
	defer cancel()
	return timedContextForInitialSync
}

// Only highly utilized objects are transformed here while others are kept as is.
func transformObject(object any) (any, error) {
	switch o := object.(type) {
	case *corev1.Pod:
		return pod.Transform(o), nil
	case *corev1.Node:
		return node.Transform(o), nil
	case *appsv1.ReplicaSet:
		return replicaset.Transform(o), nil
	case *batchv1.Job:
		return jobs.Transform(o), nil
	case *appsv1.Deployment:
		return deployment.Transform(o), nil
	case *appsv1.DaemonSet:
		return daemonset.Transform(o), nil
	case *appsv1.StatefulSet:
		return statefulset.Transform(o), nil
	case *corev1.Service:
		return service.Transform(o), nil
	}
	return object, nil
}

// setupInformer adds event handlers to informers and setups a metadataStore.
func (rw *resourceWatcher) setupInformer(gvk schema.GroupVersionKind, informer cache.SharedIndexInformer) {
	err := informer.SetTransform(transformObject)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error setting informer transform function")
	}
	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    rw.onAdd,
		UpdateFunc: rw.onUpdate,
		DeleteFunc: rw.onDelete,
	})
	if err != nil {
		log.Logger.Error().Err(err).Msg("error adding event handler to informer")
	}
	rw.metadataStore.Setup(gvk, informer.GetStore())
}

func (rw *resourceWatcher) onAdd(obj any) {
	log.Logger.Info().Msg("onAdd pre-wait")
	rw.waitForInitialInformerSync()
	log.Logger.Info().Msg("onAdd post-wait")

	rw.emitEvent(obj, metadata.EventTypeUpdate)
}

func (rw *resourceWatcher) onUpdate(oldObj, newObj any) {
	log.Logger.Info().Msg("onUpdate pre-wait")
	rw.waitForInitialInformerSync()
	log.Logger.Info().Msg("onUpdate post-wait")

	rw.emitEvent(newObj, metadata.EventTypeUpdate)
}

func (rw *resourceWatcher) onDelete(oldObj any) {
	log.Logger.Info().Msg("onDelete pre-wait")
	rw.waitForInitialInformerSync()
	log.Logger.Info().Msg("onDelete post-wait")

	rw.emitEvent(oldObj, metadata.EventTypeDelete)
}

// objMetadata returns the metadata for the given object.
func (rw *resourceWatcher) objMetadata(obj any) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	switch o := obj.(type) {
	case *corev1.Pod:
		return pod.GetMetadata(o, rw.metadataStore)
	case *corev1.Node:
		return node.GetMetadata(o)
	case *corev1.ReplicationController:
		return replicationcontroller.GetMetadata(o)
	case *appsv1.Deployment:
		return deployment.GetMetadata(o)
	case *appsv1.ReplicaSet:
		return replicaset.GetMetadata(o)
	case *appsv1.DaemonSet:
		return daemonset.GetMetadata(o)
	case *appsv1.StatefulSet:
		return statefulset.GetMetadata(o)
	case *batchv1.Job:
		return jobs.GetMetadata(o)
	case *batchv1.CronJob:
		return cronjob.GetMetadata(o)
	case *autoscalingv2.HorizontalPodAutoscaler:
		return hpa.GetMetadata(o)
	case *corev1.Namespace:
		return namespace.GetMetadata(o)
	}
	return nil
}

func (rw *resourceWatcher) waitForInitialInformerSync() {
	if rw.initialSyncDone.Load() || rw.initialSyncTimedOut.Load() {
		return
	}

	// Wait till initial sync is complete or timeout.
	for !rw.initialSyncDone.Load() {
		if rw.initialSyncTimedOut.Load() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
