package controller

import (
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/event"
	"github.com/opisvigilant/futura/watcher/internal/handlers"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/utils"

	apps_v1 "k8s.io/api/apps/v1"
	autoscaling_v1 "k8s.io/api/autoscaling/v1"
	batch_v1 "k8s.io/api/batch/v1"
	api_v1 "k8s.io/api/core/v1"
	events_v1 "k8s.io/api/events/v1"
	networking_v1 "k8s.io/api/networking/v1"
	rbac_v1 "k8s.io/api/rbac/v1"
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

// Event indicate the informerEvent
type Event struct {
	key          string
	eventType    string
	namespace    string
	resourceType string
	apiVersion   string
	obj          runtime.Object
	oldObj       runtime.Object
}

type watcherType int

const (
	PodType watcherType = iota
	CoreEventType
	EventType
	HPAType
	DaemonSetType
	StatefulSetType
	ReplicaSetType
	ServiceType
	DeploymentType
	NamespaceType
	JobType
	NodeType
	ServiceAccountType
	ClusterRoleType
	ClusterRoleBindingType
	PersistentVolumeType
	SecretType
	ConfigMapType
	IngressType
)

// Controller object
type Controller struct {
	watchers map[watcherType]*watcher
}

type watcher struct {
	informer     cache.SharedIndexInformer
	stopCh       chan struct{}
	clientset    kubernetes.Interface
	queue        workqueue.RateLimitingInterface
	eventHandler handlers.Handler
}

func New(c *config.Configuration, eventHandler handlers.Handler) (*Controller, error) {
	var kubeClient kubernetes.Interface
	if _, err := rest.InClusterConfig(); err != nil {
		kubeClient = utils.GetClientOutOfCluster()
	} else {
		kubeClient = utils.GetClient()
	}

	factory := informers.NewFilteredSharedInformerFactory(kubeClient, 0, "", nil)

	watchers := map[watcherType]*watcher{
		PodType:                newWatcher(kubeClient, factory.Core().V1().Pods().Informer(), eventHandler, objName(api_v1.Pod{}), V1),
		CoreEventType:          newWatcher(kubeClient, factory.Core().V1().Events().Informer(), eventHandler, objName(api_v1.Event{}), V1),
		EventType:              newWatcher(kubeClient, factory.Events().V1().Events().Informer(), eventHandler, objName(events_v1.Event{}), EVENTS_V1),
		HPAType:                newWatcher(kubeClient, factory.Autoscaling().V1().HorizontalPodAutoscalers().Informer(), eventHandler, objName(autoscaling_v1.HorizontalPodAutoscaler{}), AUTOSCALING_V1),
		DaemonSetType:          newWatcher(kubeClient, factory.Apps().V1().DaemonSets().Informer(), eventHandler, objName(apps_v1.DaemonSet{}), APPS_V1),
		StatefulSetType:        newWatcher(kubeClient, factory.Apps().V1().StatefulSets().Informer(), eventHandler, objName(apps_v1.StatefulSet{}), APPS_V1),
		ReplicaSetType:         newWatcher(kubeClient, factory.Apps().V1().ReplicaSets().Informer(), eventHandler, objName(apps_v1.ReplicaSet{}), APPS_V1),
		ServiceType:            newWatcher(kubeClient, factory.Core().V1().Services().Informer(), eventHandler, objName(api_v1.Service{}), V1),
		DeploymentType:         newWatcher(kubeClient, factory.Apps().V1().Deployments().Informer(), eventHandler, objName(apps_v1.Deployment{}), APPS_V1),
		NamespaceType:          newWatcher(kubeClient, factory.Core().V1().Namespaces().Informer(), eventHandler, objName(api_v1.Namespace{}), V1),
		JobType:                newWatcher(kubeClient, factory.Batch().V1().Jobs().Informer(), eventHandler, objName(batch_v1.Job{}), BATCH_V1),
		NodeType:               newWatcher(kubeClient, factory.Core().V1().Nodes().Informer(), eventHandler, objName(api_v1.Node{}), V1),
		ServiceAccountType:     newWatcher(kubeClient, factory.Core().V1().ServiceAccounts().Informer(), eventHandler, objName(api_v1.ServiceAccount{}), V1),
		ClusterRoleType:        newWatcher(kubeClient, factory.Rbac().V1().ClusterRoles().Informer(), eventHandler, objName(rbac_v1.ClusterRole{}), RBAC_V1),
		ClusterRoleBindingType: newWatcher(kubeClient, factory.Rbac().V1().ClusterRoleBindings().Informer(), eventHandler, objName(rbac_v1.ClusterRoleBinding{}), RBAC_V1),
		PersistentVolumeType:   newWatcher(kubeClient, factory.Core().V1().PersistentVolumes().Informer(), eventHandler, objName(api_v1.PersistentVolume{}), V1),
		SecretType:             newWatcher(kubeClient, factory.Core().V1().Secrets().Informer(), eventHandler, objName(api_v1.Secret{}), V1),
		ConfigMapType:          newWatcher(kubeClient, factory.Core().V1().ConfigMaps().Informer(), eventHandler, objName(api_v1.ConfigMap{}), V1),
		IngressType:            newWatcher(kubeClient, factory.Networking().V1().Ingresses().Informer(), eventHandler, objName(networking_v1.Ingress{}), NETWORKING_V1),
	}

	return &Controller{
		watchers: watchers,
	}, nil
}

func newWatcher(kubeClient kubernetes.Interface, informer cache.SharedIndexInformer, eventHandler handlers.Handler, resourceType string, apiVersion string) *watcher {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	var newEvent Event
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
			logger.Logger().Info().Fields(map[string]interface{}{
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
		informer:     informer,
		clientset:    kubeClient,
		queue:        queue,
		eventHandler: eventHandler,
		stopCh:       make(chan struct{}),
	}
}

// Start prepares watchers and run their controllers, then waits for process termination signals
func (c *Controller) Start() {
	for _, w := range c.watchers {
		defer close(w.stopCh)
		go w.run(w.stopCh)
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}

func objName(obj interface{}) string {
	return reflect.TypeOf(obj).Name()
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
	err := w.processItem(newEvent.(Event))
	if err == nil {
		// No error, reset the ratelimit counters
		w.queue.Forget(newEvent)
	} else if w.queue.NumRequeues(newEvent) < maxRetries {
		logger.Logger().Error().Msgf("error processing %s (will retry): %v", newEvent.(Event).key, err)
		w.queue.AddRateLimited(newEvent)
	} else {
		// err != nil and too many retries
		logger.Logger().Error().Msgf("error processing %s (giving up): %v", newEvent.(Event).key, err)
		w.queue.Forget(newEvent)
		utilruntime.HandleError(err)
	}
	return true
}

// TODO: Enhance event creation using client-side cacheing machanisms - pending
func (w *watcher) processItem(newEvent Event) error {
	// NOTE that obj will be nil on deletes!
	obj, _, err := w.informer.GetIndexer().GetByKey(newEvent.key)

	if err != nil {
		return fmt.Errorf("error fetching object with key %s from store: %v", newEvent.key, err)
	}
	// get object's metedata
	objectMeta := utils.GetObjectMetaData(obj)

	// hold status type for default critical alerts
	var status string

	// namespace retrived from event key incase namespace value is empty
	if newEvent.namespace == "" && strings.Contains(newEvent.key, "/") {
		substring := strings.Split(newEvent.key, "/")
		newEvent.namespace = substring[0]
		newEvent.key = substring[1]
	} else {
		newEvent.namespace = objectMeta.Namespace
	}

	// process events based on its type
	switch newEvent.eventType {
	case "create":
		// compare CreationTimestamp and serverStartTime and alert only on latest events
		// Could be Replaced by using Delta or DeltaFIFO
		if objectMeta.CreationTimestamp.Sub(serverStartTime).Seconds() > 0 {
			switch newEvent.resourceType {
			case "NodeNotReady":
				status = "Danger"
			case "NodeReady":
				status = "Normal"
			case "NodeRebooted":
				status = "Danger"
			case "Backoff":
				status = "Danger"
			default:
				status = "Normal"
			}
			kbEvent := event.Event{
				Name:       newEvent.key,
				Namespace:  newEvent.namespace,
				Kind:       newEvent.resourceType,
				ApiVersion: newEvent.apiVersion,
				Status:     status,
				Reason:     "Created",
				Timestamp:  time.Now().UnixMilli(),
				Obj:        newEvent.obj,
			}
			w.eventHandler.Handle(kbEvent)
			return nil
		}
	case "update":
		switch newEvent.resourceType {
		case "Backoff":
			status = "Danger"
		default:
			status = "Warning"
		}
		kbEvent := event.Event{
			Name:       newEvent.key,
			Namespace:  newEvent.namespace,
			Kind:       newEvent.resourceType,
			ApiVersion: newEvent.apiVersion,
			Status:     status,
			Reason:     "Updated",
			Timestamp:  time.Now().UnixMilli(),
			Obj:        newEvent.obj,
			OldObj:     newEvent.oldObj,
		}
		w.eventHandler.Handle(kbEvent)
		return nil
	case "delete":
		kbEvent := event.Event{
			Name:       newEvent.key,
			Namespace:  newEvent.namespace,
			Kind:       newEvent.resourceType,
			ApiVersion: newEvent.apiVersion,
			Timestamp:  time.Now().UnixMilli(),
			Status:     "Danger",
			Reason:     "Deleted",
			Obj:        newEvent.obj,
		}
		w.eventHandler.Handle(kbEvent)
		return nil
	}
	return nil
}
