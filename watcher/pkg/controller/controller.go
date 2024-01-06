package controller

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/opisvigilant/futura/pkg/logger"
	"github.com/opisvigilant/futura/watcher/pkg/event"
	"github.com/opisvigilant/futura/watcher/pkg/utils"
	"github.com/opisvigilant/futura/watcher/pkg/webhook"

	apps_v1 "k8s.io/api/apps/v1"
	autoscaling_v1 "k8s.io/api/autoscaling/v1"
	batch_v1 "k8s.io/api/batch/v1"
	api_v1 "k8s.io/api/core/v1"
	events_v1 "k8s.io/api/events/v1"
	networking_v1 "k8s.io/api/networking/v1"
	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
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
	eventWebhook webhook.Webhook
}

func objName(obj interface{}) string {
	return reflect.TypeOf(obj).Name()
}

// TODO: we don't need the informer to be indexed
// Start prepares watchers and run their controllers, then waits for process termination signals
func Start(eventHandler webhook.Webhook) {
	var kubeClient kubernetes.Interface

	if _, err := rest.InClusterConfig(); err != nil {
		kubeClient = utils.GetClientOutOfCluster()
	} else {
		kubeClient = utils.GetClient()
	}

	watchCoreEvent(kubeClient, eventHandler)
	watchEvent(kubeClient, eventHandler)
	watchPodEvent(kubeClient, eventHandler)
	watchHPAEvent(kubeClient, eventHandler)
	watchDaemonSetEvent(kubeClient, eventHandler)
	watchStatefulSetEvent(kubeClient, eventHandler)
	watchReplicaSetEvent(kubeClient, eventHandler)
	watchServicesEvent(kubeClient, eventHandler)
	watchDeploymentEvent(kubeClient, eventHandler)
	watchNamespaceEvent(kubeClient, eventHandler)
	watchReplicationControllerEvent(kubeClient, eventHandler)
	watchJobEvent(kubeClient, eventHandler)
	watchNodeEvent(kubeClient, eventHandler)
	watchServiceAccountEvent(kubeClient, eventHandler)
	watchClusterRoleEvent(kubeClient, eventHandler)
	watchClusterRoleBindingEvent(kubeClient, eventHandler)
	watchPersistentVolumeEvent(kubeClient, eventHandler)
	watchSecretEvent(kubeClient, eventHandler)
	watchConfigmapEvent(kubeClient, eventHandler)
	watchIngressEvent(kubeClient, eventHandler)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}

func watchCoreEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	allCoreEventsInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = ""
				return kubeClient.CoreV1().Events("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = ""
				return kubeClient.CoreV1().Events("").Watch(context.Background(), options)
			},
		},
		&api_v1.Event{},
		0, //Skip resync
		cache.Indexers{},
	)

	allCoreEventsController := newResourceController(kubeClient, eventHandler, allCoreEventsInformer, objName(api_v1.Event{}), V1)
	stopAllCoreEventsCh := make(chan struct{})
	defer close(stopAllCoreEventsCh)

	go allCoreEventsController.Run(stopAllCoreEventsCh)
}

func watchEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	allEventsInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = ""
				return kubeClient.EventsV1().Events("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = ""
				return kubeClient.EventsV1().Events("").Watch(context.Background(), options)
			},
		},
		&events_v1.Event{},
		0, //Skip resync
		cache.Indexers{},
	)

	allEventsController := newResourceController(kubeClient, eventHandler, allEventsInformer, objName(events_v1.Event{}), EVENTS_V1)
	stopAllEventsCh := make(chan struct{})
	defer close(stopAllEventsCh)

	go allEventsController.Run(stopAllEventsCh)
}

func watchIngressEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.NetworkingV1().Ingresses("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.NetworkingV1().Ingresses("").Watch(context.Background(), options)
			},
		},
		&networking_v1.Ingress{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(networking_v1.Ingress{}), NETWORKING_V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchConfigmapEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().ConfigMaps("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().ConfigMaps("").Watch(context.Background(), options)
			},
		},
		&api_v1.ConfigMap{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.ConfigMap{}), V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchSecretEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().Secrets("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().Secrets("").Watch(context.Background(), options)
			},
		},
		&api_v1.Secret{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.Secret{}), V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchPersistentVolumeEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().PersistentVolumes().List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().PersistentVolumes().Watch(context.Background(), options)
			},
		},
		&api_v1.PersistentVolume{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.PersistentVolume{}), V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchClusterRoleBindingEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.RbacV1().ClusterRoleBindings().List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.RbacV1().ClusterRoleBindings().Watch(context.Background(), options)
			},
		},
		&rbac_v1.ClusterRoleBinding{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(rbac_v1.ClusterRoleBinding{}), RBAC_V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchClusterRoleEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.RbacV1().ClusterRoles().List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.RbacV1().ClusterRoles().Watch(context.Background(), options)
			},
		},
		&rbac_v1.ClusterRole{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(rbac_v1.ClusterRole{}), RBAC_V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchServiceAccountEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().ServiceAccounts("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().ServiceAccounts("").Watch(context.Background(), options)
			},
		},
		&api_v1.ServiceAccount{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.ServiceAccount{}), V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchNodeEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().Nodes().List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().Nodes().Watch(context.Background(), options)
			},
		},
		&api_v1.Node{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.Node{}), V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchJobEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.BatchV1().Jobs("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.BatchV1().Jobs("").Watch(context.Background(), options)
			},
		},
		&batch_v1.Job{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(batch_v1.Job{}), BATCH_V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchReplicationControllerEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().ReplicationControllers("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().ReplicationControllers("").Watch(context.Background(), options)
			},
		},
		&api_v1.ReplicationController{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.ReplicationController{}), V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchNamespaceEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().Namespaces().List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().Namespaces().Watch(context.Background(), options)
			},
		},
		&api_v1.Namespace{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.Namespace{}), V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchDeploymentEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.AppsV1().Deployments("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.AppsV1().Deployments("").Watch(context.Background(), options)
			},
		},
		&apps_v1.Deployment{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(apps_v1.Deployment{}), APPS_V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchServicesEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().Services("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().Services("").Watch(context.Background(), options)
			},
		},
		&api_v1.Service{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.Service{}), V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchReplicaSetEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.AppsV1().ReplicaSets("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.AppsV1().ReplicaSets("").Watch(context.Background(), options)
			},
		},
		&apps_v1.ReplicaSet{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(apps_v1.ReplicaSet{}), APPS_V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchStatefulSetEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.AppsV1().StatefulSets("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.AppsV1().StatefulSets("").Watch(context.Background(), options)
			},
		},
		&apps_v1.StatefulSet{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(apps_v1.StatefulSet{}), APPS_V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchDaemonSetEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.AppsV1().DaemonSets("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.AppsV1().DaemonSets("").Watch(context.Background(), options)
			},
		},
		&apps_v1.DaemonSet{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(apps_v1.DaemonSet{}), APPS_V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchHPAEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.AutoscalingV1().HorizontalPodAutoscalers("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.AutoscalingV1().HorizontalPodAutoscalers("").Watch(context.Background(), options)
			},
		},
		&autoscaling_v1.HorizontalPodAutoscaler{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(autoscaling_v1.HorizontalPodAutoscaler{}), AUTOSCALING_V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func watchPodEvent(kubeClient kubernetes.Interface, eventHandler webhook.Webhook) {
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return kubeClient.CoreV1().Pods("").List(context.Background(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return kubeClient.CoreV1().Pods("").Watch(context.Background(), options)
			},
		},
		&api_v1.Pod{},
		0, //Skip resync
		cache.Indexers{},
	)

	c := newResourceController(kubeClient, eventHandler, informer, objName(api_v1.Pod{}), V1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	go c.Run(stopCh)
}

func newResourceController(client kubernetes.Interface, eventHandler webhook.Webhook, informer cache.SharedIndexInformer, resourceType string, apiVersion string) *Controller {
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
					"pkg": "kubewatch-" + resourceType,
				}).Msgf("cannot convert to runtime.Object for add on %v", obj)
			}
			logger.Logger().Info().Fields(map[string]interface{}{
				"pkg": "kubewatch-" + resourceType,
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
					"pkg": "kubewatch-" + resourceType,
				}).Msgf("cannot convert to runtime.Object for update on %v", new)
			}
			newEvent.oldObj, ok = old.(runtime.Object)
			if !ok {
				logger.Logger().Error().Fields(map[string]interface{}{
					"pkg": "kubewatch-" + resourceType,
				}).Msgf("cannot convert old to runtime.Object for update on %v", old)
			}
			logger.Logger().Info().Fields(map[string]interface{}{
				"pkg": "kubewatch-" + resourceType,
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
					"pkg": "kubewatch-" + resourceType,
				}).Msgf("cannot convert to runtime.Object for delete on %v", obj)
			}
			logger.Logger().Info().Fields(map[string]interface{}{
				"pkg": "kubewatch-" + resourceType,
			}).Msgf("Processing delete to %v: %s", resourceType, newEvent.key)
			if err == nil {
				queue.Add(newEvent)
			}
		},
	})

	return &Controller{
		clientset:    client,
		informer:     informer,
		queue:        queue,
		eventWebhook: eventHandler,
	}
}

// Run starts the kubewatch controller
func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	logger.Logger().Info().Msg("Starting kubewatch controller")
	serverStartTime = time.Now().Local()

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	logger.Logger().Info().Msg("Kubewatch controller synced and ready")

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
		logger.Logger().Error().Msgf("Error processing %s (will retry): %v", newEvent.(Event).key, err)
		c.queue.AddRateLimited(newEvent)
	} else {
		// err != nil and too many retries
		logger.Logger().Error().Msgf("Error processing %s (giving up): %v", newEvent.(Event).key, err)
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
		return fmt.Errorf("Error fetching object with key %s from store: %v", newEvent.key, err)
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
			c.eventWebhook.Handle(kbEvent)
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
		c.eventWebhook.Handle(kbEvent)
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
		c.eventWebhook.Handle(kbEvent)
		return nil
	}
	return nil
}
