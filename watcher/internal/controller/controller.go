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

// Controller object
type Controller struct {
	clientset    kubernetes.Interface
	queue        workqueue.RateLimitingInterface
	informer     cache.SharedIndexInformer
	eventHandler handlers.Handler
}

func objName(obj interface{}) string {
	return reflect.TypeOf(obj).Name()
}

// TODO: we don't need the informer to be indexed
// Start prepares watchers and run their controllers, then waits for process termination signals
func Start(c *config.Configuration, eventHandler handlers.Handler) {
	var kubeClient kubernetes.Interface
	if _, err := rest.InClusterConfig(); err != nil {
		kubeClient = utils.GetClientOutOfCluster()
	} else {
		kubeClient = utils.GetClient()
	}

	factory := informers.NewFilteredSharedInformerFactory(kubeClient, 0, "", nil)

	// Pod informer
	{
		informer := factory.Core().V1().Pods().Informer()
		ctrl := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.Pod{}), V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go ctrl.Run(stopCh)
	}

	// CoreEvent informer
	{
		informer := factory.Core().V1().Events().Informer()

		allCoreEventsController := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.Event{}), V1)
		stopAllCoreEventsCh := make(chan struct{})
		defer close(stopAllCoreEventsCh)

		go allCoreEventsController.Run(stopAllCoreEventsCh)
	}

	// Event informer
	{
		informer := factory.Events().V1().Events().Informer()

		allEventsController := newResourceController(kubeClient, eventHandler, informer, objName(events_v1.Event{}), EVENTS_V1)
		stopAllEventsCh := make(chan struct{})
		defer close(stopAllEventsCh)

		go allEventsController.Run(stopAllEventsCh)
	}

	// HPA informer
	{
		informer := factory.Autoscaling().V1().HorizontalPodAutoscalers().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(autoscaling_v1.HorizontalPodAutoscaler{}), AUTOSCALING_V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Daemonset informer
	{
		informer := factory.Apps().V1().DaemonSets().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(apps_v1.DaemonSet{}), APPS_V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Statefulset informer
	{
		informer := factory.Apps().V1().StatefulSets().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(apps_v1.StatefulSet{}), APPS_V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Replicaset informer
	{
		informer := factory.Apps().V1().ReplicaSets().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(apps_v1.ReplicaSet{}), APPS_V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Service informer
	{
		informer := factory.Core().V1().Services().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.Service{}), V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Deployment informer
	{
		informer := factory.Apps().V1().Deployments().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(apps_v1.Deployment{}), APPS_V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Namespace informer
	{
		informer := factory.Core().V1().Namespaces().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.Namespace{}), V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Replicaset informer
	{
		informer := factory.Apps().V1().ReplicaSets().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.ReplicationController{}), V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Job informer
	{
		informer := factory.Batch().V1().Jobs().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(batch_v1.Job{}), BATCH_V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Node informer
	{
		informer := factory.Core().V1().Nodes().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.Node{}), V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// ServiceAccount informer
	{
		informer := factory.Core().V1().ServiceAccounts().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.ServiceAccount{}), V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// ClusterRole informer
	{
		informer := factory.Rbac().V1().ClusterRoles().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(rbac_v1.ClusterRole{}), RBAC_V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// ClusterRoleBinding informer
	{
		informer := factory.Rbac().V1().ClusterRoleBindings().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(rbac_v1.ClusterRoleBinding{}), RBAC_V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// PersistenVolume informer
	{
		informer := factory.Core().V1().PersistentVolumes().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.PersistentVolume{}), V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Secret informer
	{
		informer := factory.Core().V1().Secrets().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.Secret{}), V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Configmap informer
	{
		informer := factory.Core().V1().ConfigMaps().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.ConfigMap{}), V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	// Ingress informer
	{
		informer := factory.Networking().V1().Ingresses().Informer()

		c := newResourceController(kubeClient, eventHandler, informer, objName(networking_v1.Ingress{}), NETWORKING_V1)
		stopCh := make(chan struct{})
		defer close(stopCh)

		go c.Run(stopCh)
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}

func newResourceController(client kubernetes.Interface, eventHandler handlers.Handler, informer cache.SharedIndexInformer, resourceType string, apiVersion string) *Controller {
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

	return &Controller{
		clientset:    client,
		informer:     informer,
		queue:        queue,
		eventHandler: eventHandler,
	}
}

// Run starts the watcher controller
func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	logger.Logger().Info().Msg("starting watcher controller")
	serverStartTime = time.Now().Local()

	go c.informer.Run(stopCh)

	if !cache.WaitForNamedCacheSync("watcher", stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	logger.Logger().Info().Msg("watcher controller synced and ready")

	wait.Until(c.runWorker, time.Second, stopCh)
}

// HasSynced is required for the cache.Controller interface.
func (c *Controller) HasSynced() bool {
	return c.informer.HasSynced()
}

// LastSyncResourceVersion is required for the cache.Controller interface.
func (c *Controller) LastSyncResourceVersion() string {
	return c.informer.LastSyncResourceVersion()
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
		// continue looping
	}
}

func (c *Controller) processNextItem() bool {
	newEvent, quit := c.queue.Get()

	if quit {
		return false
	}
	defer c.queue.Done(newEvent)
	err := c.processItem(newEvent.(Event))
	if err == nil {
		// No error, reset the ratelimit counters
		c.queue.Forget(newEvent)
	} else if c.queue.NumRequeues(newEvent) < maxRetries {
		logger.Logger().Error().Msgf("error processing %s (will retry): %v", newEvent.(Event).key, err)
		c.queue.AddRateLimited(newEvent)
	} else {
		// err != nil and too many retries
		logger.Logger().Error().Msgf("error processing %s (giving up): %v", newEvent.(Event).key, err)
		c.queue.Forget(newEvent)
		utilruntime.HandleError(err)
	}
	return true
}

// TODO: Enhance event creation using client-side cacheing machanisms - pending
func (c *Controller) processItem(newEvent Event) error {
	// NOTE that obj will be nil on deletes!
	obj, _, err := c.informer.GetIndexer().GetByKey(newEvent.key)

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
				Obj:        newEvent.obj,
			}
			c.eventHandler.Handle(kbEvent)
			return nil
		}
	case "update":
		/* TODOs
		- enahace update event processing in such a way that, it send alerts about what got changed.
		*/
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
			Obj:        newEvent.obj,
			OldObj:     newEvent.oldObj,
		}
		c.eventHandler.Handle(kbEvent)
		return nil
	case "delete":
		kbEvent := event.Event{
			Name:       newEvent.key,
			Namespace:  newEvent.namespace,
			Kind:       newEvent.resourceType,
			ApiVersion: newEvent.apiVersion,
			Status:     "Danger",
			Reason:     "Deleted",
			Obj:        newEvent.obj,
		}
		c.eventHandler.Handle(kbEvent)
		return nil
	}
	return nil
}
