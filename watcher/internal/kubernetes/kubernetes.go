package kubernetes

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/utils"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const maxRetries = 5
const V1 = "v1"
const AUTOSCALING_V1 = "autoscaling/v1"
const APPS_V1 = "apps/v1"
const BATCH_V1 = "batch/v1"
const RBAC_V1 = "rbac.authorization.k8s.io/v1"
const NETWORKING_V1 = "networking.k8s.io/v1"
const EVENTS_V1 = "events.k8s.io/v1"

var serverStartTime time.Time

type watcherType string

const (
	PodType                watcherType = "POD"
	CoreEventType          watcherType = "CORE_EVENT"
	EventType              watcherType = "EVENT"
	HPAType                watcherType = "HPA"
	DaemonSetType          watcherType = "DAEMONSET"
	StatefulSetType        watcherType = "STATEFULSET"
	ReplicaSetType         watcherType = "REPLICASET"
	ServiceType            watcherType = "SERVICE"
	DeploymentType         watcherType = "DEPLOYMENT"
	NamespaceType          watcherType = "NAMESPACE"
	JobType                watcherType = "JOB"
	NodeType               watcherType = "NODE"
	ServiceAccountType     watcherType = "SERVICE_ACCOUNT"
	ClusterRoleType        watcherType = "CLUSTER_ROLE"
	ClusterRoleBindingType watcherType = "CLUSTER_ROLE_BINDING"
	PersistentVolumeType   watcherType = "PERSISTENT_VOLUME"
	SecretType             watcherType = "SECRET"
	ConfigMapType          watcherType = "CONFIGMAP"
	IngressType            watcherType = "INGRESS"
)

// InformerEvent indicate the informerEvent
type InformerEvent struct {
	key          string
	eventType    string
	namespace    string
	resourceType watcherType
	apiVersion   string
	obj          runtime.Object
	oldObj       runtime.Object
}

// KubernetesCollector object
type KubernetesCollector struct {
	watchers map[watcherType]*watcher
}

type watcher struct {
	informer  cache.SharedIndexInformer
	stopCh    chan struct{}
	clientset kubernetes.Interface
	queue     workqueue.RateLimitingInterface
	events    chan interface{}
}

func New(c *config.Configuration, events chan interface{}) (*KubernetesCollector, error) {
	var kubeClient kubernetes.Interface
	if _, err := rest.InClusterConfig(); err != nil {
		kubeClient = utils.GetClientOutOfCluster()
	} else {
		kubeClient = utils.GetClient()
	}

	factory := informers.NewFilteredSharedInformerFactory(kubeClient, 0, "", nil)

	watchers := map[watcherType]*watcher{
		PodType:              newWatcher(kubeClient, factory.Core().V1().Pods().Informer(), events, PodType, V1),
		CoreEventType:        newWatcher(kubeClient, factory.Core().V1().Events().Informer(), events, CoreEventType, V1),
		EventType:            newWatcher(kubeClient, factory.Events().V1().Events().Informer(), events, EventType, EVENTS_V1),
		HPAType:              newWatcher(kubeClient, factory.Autoscaling().V1().HorizontalPodAutoscalers().Informer(), events, HPAType, AUTOSCALING_V1),
		DaemonSetType:        newWatcher(kubeClient, factory.Apps().V1().DaemonSets().Informer(), events, DaemonSetType, APPS_V1),
		StatefulSetType:      newWatcher(kubeClient, factory.Apps().V1().StatefulSets().Informer(), events, StatefulSetType, APPS_V1),
		ReplicaSetType:       newWatcher(kubeClient, factory.Apps().V1().ReplicaSets().Informer(), events, ReplicaSetType, APPS_V1),
		ServiceType:          newWatcher(kubeClient, factory.Core().V1().Services().Informer(), events, ServiceType, V1),
		DeploymentType:       newWatcher(kubeClient, factory.Apps().V1().Deployments().Informer(), events, DeploymentType, APPS_V1),
		NamespaceType:        newWatcher(kubeClient, factory.Core().V1().Namespaces().Informer(), events, NamespaceType, V1),
		JobType:              newWatcher(kubeClient, factory.Batch().V1().Jobs().Informer(), events, JobType, BATCH_V1),
		NodeType:             newWatcher(kubeClient, factory.Core().V1().Nodes().Informer(), events, NodeType, V1),
		PersistentVolumeType: newWatcher(kubeClient, factory.Core().V1().PersistentVolumes().Informer(), events, PersistentVolumeType, V1),
		IngressType:          newWatcher(kubeClient, factory.Networking().V1().Ingresses().Informer(), events, IngressType, NETWORKING_V1),
		// ServiceAccountType:     newWatcher(kubeClient, factory.Core().V1().ServiceAccounts().Informer(), eventHandler, ServiceAccountType, V1),
		// ClusterRoleType:        newWatcher(kubeClient, factory.Rbac().V1().ClusterRoles().Informer(), eventHandler, ClusterRoleType, RBAC_V1),
		// ClusterRoleBindingType: newWatcher(kubeClient, factory.Rbac().V1().ClusterRoleBindings().Informer(), eventHandler, ClusterRoleBindingType, RBAC_V1),
		// SecretType:             newWatcher(kubeClient, factory.Core().V1().Secrets().Informer(), eventHandler, SecretType, V1),
		// ConfigMapType:          newWatcher(kubeClient, factory.Core().V1().ConfigMaps().Informer(), eventHandler, ConfigMapType, V1),
	}

	return &KubernetesCollector{
		watchers: watchers,
	}, nil
}

func newWatcher(kubeClient kubernetes.Interface, informer cache.SharedIndexInformer, events chan interface{}, resourceType watcherType, apiVersion string) *watcher {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	var newEvent InformerEvent
	var err error

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			var ok bool
			newEvent.namespace = "" // namespace retrived in processItem incase namespace value is empty
			newEvent.key, err = cache.MetaNamespaceKeyFunc(obj)
			newEvent.eventType = "create"
			newEvent.resourceType = resourceType
			newEvent.apiVersion = apiVersion
			newEvent.obj, ok = obj.(runtime.Object)
			if !ok {
				logger.Logger().Error().Fields(map[string]interface{}{
					"pkg": "watcher-" + resourceType,
				}).Msgf("cannot convert to runtime.Object for add on %v", obj)
			}
			logger.Logger().Info().Fields(map[string]interface{}{
				"pkg": "watcher-" + resourceType,
			}).Msgf("Processing add to %v: %s", resourceType, newEvent.key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			var ok bool
			newEvent.namespace = "" // namespace retrived in processItem incase namespace value is empty
			newEvent.key, err = cache.MetaNamespaceKeyFunc(old)
			newEvent.eventType = "update"
			newEvent.resourceType = resourceType
			newEvent.apiVersion = apiVersion
			newEvent.obj, ok = new.(runtime.Object)
			if !ok {
				logger.Logger().Error().Fields(map[string]interface{}{
					"pkg": "watcher-" + resourceType,
				}).Msgf("cannot convert to runtime.Object for update on %v", new)
			}
			newEvent.oldObj, ok = old.(runtime.Object)
			if !ok {
				logger.Logger().Error().Fields(map[string]interface{}{
					"pkg": "watcher-" + resourceType,
				}).Msgf("cannot convert old to runtime.Object for update on %v", old)
			}
			logger.Logger().Debug().Fields(map[string]interface{}{
				"pkg": "watcher-" + resourceType,
			}).Msgf("Processing update to %v: %s", resourceType, newEvent.key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
		DeleteFunc: func(obj interface{}) {
			var ok bool
			newEvent.namespace = "" // namespace retrived in processItem incase namespace value is empty
			newEvent.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			newEvent.eventType = "delete"
			newEvent.resourceType = resourceType
			newEvent.apiVersion = apiVersion
			newEvent.obj, ok = obj.(runtime.Object)
			if !ok {
				logger.Logger().Error().Fields(map[string]interface{}{
					"pkg": "watcher-" + resourceType,
				}).Msgf("cannot convert to runtime.Object for delete on %v", obj)
			}
			logger.Logger().Info().Fields(map[string]interface{}{
				"pkg": "watcher-" + resourceType,
			}).Msgf("processing delete to %v: %s", resourceType, newEvent.key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
	})

	return &watcher{
		informer:  informer,
		clientset: kubeClient,
		queue:     queue,
		events:    events,
		stopCh:    make(chan struct{}),
	}
}

// Start prepares watchers and run their controllers, then waits for process termination signals
func (c *KubernetesCollector) Start() {
	for _, w := range c.watchers {
		defer close(w.stopCh)
		go w.run(w.stopCh)
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}

// run starts the watcher controller
func (w *watcher) run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer w.queue.ShutDown()

	logger.Logger().Info().Msg("starting watcher controller")
	serverStartTime = time.Now().Local()

	go w.informer.Run(stopCh)

	if !cache.WaitForNamedCacheSync("watcher", stopCh, w.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	logger.Logger().Info().Msg("watcher controller synced and ready")

	wait.Until(w.runWorker, time.Second, stopCh)
}

// HasSynced is required for the cache.Controller interface.
func (w *watcher) HasSynced() bool {
	return w.informer.HasSynced()
}

// LastSyncResourceVersion is required for the cache.Controller interface.
func (w *watcher) LastSyncResourceVersion() string {
	return w.informer.LastSyncResourceVersion()
}

func (w *watcher) runWorker() {
	for w.processNextItem() {
		// continue looping
	}
}

func (w *watcher) processNextItem() bool {
	newEvent, quit := w.queue.Get()
	if quit {
		return false
	}

	defer w.queue.Done(newEvent)
	if err := w.processItem(newEvent.(InformerEvent)); err == nil {
		// No error, reset the ratelimit counters
		w.queue.Forget(newEvent)
	} else if w.queue.NumRequeues(newEvent) < maxRetries {
		logger.Logger().Error().Msgf("error processing %s (will retry): %v", newEvent.(InformerEvent).key, err)
		w.queue.AddRateLimited(newEvent)
	} else {
		// err != nil and too many retries
		logger.Logger().Error().Msgf("error processing %s (giving up): %v", newEvent.(InformerEvent).key, err)
		w.queue.Forget(newEvent)
		utilruntime.HandleError(err)
	}
	return true
}

type triggerType string

const (
	CreateType triggerType = "CREATE"
	UpdateType triggerType = "UPDATE"
	DeleteType triggerType = "DELETE"
)

// TODO: Enhance event creation using client-side cacheing machanisms - pending
func (w *watcher) processItem(newEvent InformerEvent) error {
	// NOTE that obj will be nil on deletes!
	obj, _, err := w.informer.GetIndexer().GetByKey(newEvent.key)

	if err != nil {
		return fmt.Errorf("error fetching object with key %s from store: %v", newEvent.key, err)
	}
	// get object's metedata
	objectMeta := utils.GetObjectMetaData(obj)

	// namespace retrived from event key incase namespace value is empty
	if newEvent.namespace == "" && strings.Contains(newEvent.key, "/") {
		substring := strings.Split(newEvent.key, "/")
		newEvent.namespace = substring[0]
		newEvent.key = substring[1]
	} else {
		newEvent.namespace = objectMeta.Namespace
	}

	// process events based on its type
	var tt triggerType
	switch newEvent.eventType {
	case "create":
		// compare CreationTimestamp and serverStartTime and alert only on latest events
		// Could be Replaced by using Delta or DeltaFIFO
		if objectMeta.CreationTimestamp.Sub(serverStartTime).Seconds() > 0 {
			tt = CreateType
		}
	case "update":
		tt = UpdateType
	case "delete":
		tt = DeleteType
	}
	w.events <- Event{
		Kind:        newEvent.resourceType,
		TriggetType: tt,
		Obj:         newEvent.obj,
	}
	return nil
}
