package kubernetes

// const maxRetries = 5
// const V1 = "v1"
// const AUTOSCALING_V1 = "autoscaling/v1"
// const APPS_V1 = "apps/v1"
// const BATCH_V1 = "batch/v1"
// const RBAC_V1 = "rbac.authorization.k8s.io/v1"
// const NETWORKING_V1 = "networking.k8s.io/v1"
// const EVENTS_V1 = "events.k8s.io/v1"

// var serverStartTime time.Time

// type watcherType string

// const (
// 	PodType                watcherType = "POD"
// 	CoreEventType          watcherType = "CORE_EVENT"
// 	EventType              watcherType = "EVENT"
// 	HPAType                watcherType = "HPA"
// 	DaemonSetType          watcherType = "DAEMONSET"
// 	StatefulSetType        watcherType = "STATEFULSET"
// 	ReplicaSetType         watcherType = "REPLICASET"
// 	ServiceType            watcherType = "SERVICE"
// 	DeploymentType         watcherType = "DEPLOYMENT"
// 	NamespaceType          watcherType = "NAMESPACE"
// 	JobType                watcherType = "JOB"
// 	NodeType               watcherType = "NODE"
// 	ServiceAccountType     watcherType = "SERVICE_ACCOUNT"
// 	ClusterRoleType        watcherType = "CLUSTER_ROLE"
// 	ClusterRoleBindingType watcherType = "CLUSTER_ROLE_BINDING"
// 	PersistentVolumeType   watcherType = "PERSISTENT_VOLUME"
// 	SecretType             watcherType = "SECRET"
// 	ConfigMapType          watcherType = "CONFIGMAP"
// 	IngressType            watcherType = "INGRESS"
// )

// InformerEvent indicate the informerEvent
// type InformerEvent struct {
// 	key          string
// 	eventType    string
// 	namespace    string
// 	resourceType watcherType
// 	apiVersion   string
// 	obj          runtime.Object
// 	oldObj       runtime.Object
// }

// KubernetesCollector object
// type KubernetesCollector struct {
// 	watchers map[watcherType]*watcher
// }

// type watcher struct {
// 	informer  cache.SharedIndexInformer
// 	stopCh    chan struct{}
// 	clientset kubernetes.Interface
// 	queue     workqueue.RateLimitingInterface
// 	events    chan any
// }

// func New(c *config.Configuration, events chan any) (*KubernetesCollector, error) {
// 	var kubeClient kubernetes.Interface
// 	if _, err := rest.InClusterConfig(); err != nil {
// 		kubeClient = utils.GetClientOutOfCluster()
// 	} else {
// 		kubeClient = utils.GetClient()
// 	}

// 	factory := informers.NewFilteredSharedInformerFactory(kubeClient, 0, "", nil)

// 	watchers := map[watcherType]*watcher{
// 		PodType:              newWatcher(kubeClient, factory.Core().V1().Pods().Informer(), events, PodType, V1),
// 		CoreEventType:        newWatcher(kubeClient, factory.Core().V1().Events().Informer(), events, CoreEventType, V1),
// 		EventType:            newWatcher(kubeClient, factory.Events().V1().Events().Informer(), events, EventType, EVENTS_V1),
// 		HPAType:              newWatcher(kubeClient, factory.Autoscaling().V1().HorizontalPodAutoscalers().Informer(), events, HPAType, AUTOSCALING_V1),
// 		DaemonSetType:        newWatcher(kubeClient, factory.Apps().V1().DaemonSets().Informer(), events, DaemonSetType, APPS_V1),
// 		StatefulSetType:      newWatcher(kubeClient, factory.Apps().V1().StatefulSets().Informer(), events, StatefulSetType, APPS_V1),
// 		ReplicaSetType:       newWatcher(kubeClient, factory.Apps().V1().ReplicaSets().Informer(), events, ReplicaSetType, APPS_V1),
// 		ServiceType:          newWatcher(kubeClient, factory.Core().V1().Services().Informer(), events, ServiceType, V1),
// 		DeploymentType:       newWatcher(kubeClient, factory.Apps().V1().Deployments().Informer(), events, DeploymentType, APPS_V1),
// 		NamespaceType:        newWatcher(kubeClient, factory.Core().V1().Namespaces().Informer(), events, NamespaceType, V1),
// 		JobType:              newWatcher(kubeClient, factory.Batch().V1().Jobs().Informer(), events, JobType, BATCH_V1),
// 		NodeType:             newWatcher(kubeClient, factory.Core().V1().Nodes().Informer(), events, NodeType, V1),
// 		PersistentVolumeType: newWatcher(kubeClient, factory.Core().V1().PersistentVolumes().Informer(), events, PersistentVolumeType, V1),
// 		IngressType:          newWatcher(kubeClient, factory.Networking().V1().Ingresses().Informer(), events, IngressType, NETWORKING_V1),
// 		// ServiceAccountType:     newWatcher(kubeClient, factory.Core().V1().ServiceAccounts().Informer(), eventHandler, ServiceAccountType, V1),
// 		// ClusterRoleType:        newWatcher(kubeClient, factory.Rbac().V1().ClusterRoles().Informer(), eventHandler, ClusterRoleType, RBAC_V1),
// 		// ClusterRoleBindingType: newWatcher(kubeClient, factory.Rbac().V1().ClusterRoleBindings().Informer(), eventHandler, ClusterRoleBindingType, RBAC_V1),
// 		// SecretType:             newWatcher(kubeClient, factory.Core().V1().Secrets().Informer(), eventHandler, SecretType, V1),
// 		// ConfigMapType:          newWatcher(kubeClient, factory.Core().V1().ConfigMaps().Informer(), eventHandler, ConfigMapType, V1),
// 	}

// 	return &KubernetesCollector{
// 		watchers: watchers,
// 	}, nil
// }

// func newWatcher(kubeClient kubernetes.Interface, informer cache.SharedIndexInformer, events chan any, resourceType watcherType, apiVersion string) *watcher {
// 	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
// 	var newEvent InformerEvent
// 	var err error

// 	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
// 		AddFunc: func(obj any) {
// 			var ok bool
// 			newEvent.namespace = "" // namespace retrived in processItem in case namespace value is empty
// 			newEvent.key, err = cache.MetaNamespaceKeyFunc(obj)
// 			newEvent.eventType = "create"
// 			newEvent.resourceType = resourceType
// 			newEvent.apiVersion = apiVersion
// 			newEvent.obj, ok = obj.(runtime.Object)
// 			if !ok {
// 				logger.Logger().Error().Fields(map[string]any{
// 					"pkg": "watcher-" + resourceType,
// 				}).Msgf("cannot convert to runtime.Object for add on %v", obj)
// 			}
// 			logger.Logger().Info().Fields(map[string]any{
// 				"pkg": "watcher-" + resourceType,
// 			}).Msgf("Processing add to %v: %s", resourceType, newEvent.key)
// 			if err == nil {
// 				queue.Add(newEvent)
// 			}
// 		},
// 		UpdateFunc: func(old, new any) {
// 			var ok bool
// 			newEvent.namespace = "" // namespace retrived in processItem in case namespace value is empty
// 			newEvent.key, err = cache.MetaNamespaceKeyFunc(old)
// 			newEvent.eventType = "update"
// 			newEvent.resourceType = resourceType
// 			newEvent.apiVersion = apiVersion
// 			newEvent.obj, ok = new.(runtime.Object)
// 			if !ok {
// 				logger.Logger().Error().Fields(map[string]any{
// 					"pkg": "watcher-" + resourceType,
// 				}).Msgf("cannot convert to runtime.Object for update on %v", new)
// 			}
// 			newEvent.oldObj, ok = old.(runtime.Object)
// 			if !ok {
// 				logger.Logger().Error().Fields(map[string]any{
// 					"pkg": "watcher-" + resourceType,
// 				}).Msgf("cannot convert old to runtime.Object for update on %v", old)
// 			}
// 			logger.Logger().Debug().Fields(map[string]any{
// 				"pkg": "watcher-" + resourceType,
// 			}).Msgf("Processing update to %v: %s", resourceType, newEvent.key)
// 			if err == nil {
// 				queue.Add(newEvent)
// 			}
// 		},
// 		DeleteFunc: func(obj any) {
// 			var ok bool
// 			newEvent.namespace = "" // namespace retrived in processItem in case namespace value is empty
// 			newEvent.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
// 			newEvent.eventType = "delete"
// 			newEvent.resourceType = resourceType
// 			newEvent.apiVersion = apiVersion
// 			newEvent.obj, ok = obj.(runtime.Object)
// 			if !ok {
// 				logger.Logger().Error().Fields(map[string]any{
// 					"pkg": "watcher-" + resourceType,
// 				}).Msgf("cannot convert to runtime.Object for delete on %v", obj)
// 			}
// 			logger.Logger().Info().Fields(map[string]any{
// 				"pkg": "watcher-" + resourceType,
// 			}).Msgf("processing delete to %v: %s", resourceType, newEvent.key)
// 			if err == nil {
// 				queue.Add(newEvent)
// 			}
// 		},
// 	})

// 	return &watcher{
// 		informer:  informer,
// 		clientset: kubeClient,
// 		queue:     queue,
// 		events:    events,
// 		stopCh:    make(chan struct{}),
// 	}
// }

// // Start prepares watchers and run their controllers, then waits for process termination signals
// func (c *KubernetesCollector) Start() {
// 	for _, w := range c.watchers {
// 		defer close(w.stopCh)
// 		go w.run(w.stopCh)
// 	}

// 	sigterm := make(chan os.Signal, 1)
// 	signal.Notify(sigterm, syscall.SIGTERM)
// 	signal.Notify(sigterm, syscall.SIGINT)
// 	<-sigterm
// }

// // run starts the watcher controller
// func (w *watcher) run(stopCh <-chan struct{}) {
// 	defer utilruntime.HandleCrash()
// 	defer w.queue.ShutDown()

// 	logger.Logger().Info().Msg("starting watcher controller")
// 	serverStartTime = time.Now().Local()

// 	go w.informer.Run(stopCh)

// 	if !cache.WaitForNamedCacheSync("watcher", stopCh, w.HasSynced) {
// 		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
// 		return
// 	}

// 	logger.Logger().Info().Msg("watcher controller synced and ready")

// 	wait.Until(w.runWorker, time.Second, stopCh)
// }

// // HasSynced is required for the cache.Controller interface.
// func (w *watcher) HasSynced() bool {
// 	return w.informer.HasSynced()
// }

// // LastSyncResourceVersion is required for the cache.Controller interface.
// func (w *watcher) LastSyncResourceVersion() string {
// 	return w.informer.LastSyncResourceVersion()
// }

// func (w *watcher) runWorker() {
// 	for w.processNextItem() {
// 		// continue looping
// 	}
// }

// func (w *watcher) processNextItem() bool {
// 	newEvent, quit := w.queue.Get()
// 	if quit {
// 		return false
// 	}

// 	defer w.queue.Done(newEvent)
// 	if err := w.processItem(newEvent.(InformerEvent)); err == nil {
// 		// No error, reset the ratelimit counters
// 		w.queue.Forget(newEvent)
// 	} else if w.queue.NumRequeues(newEvent) < maxRetries {
// 		logger.Logger().Error().Msgf("error processing %s (will retry): %v", newEvent.(InformerEvent).key, err)
// 		w.queue.AddRateLimited(newEvent)
// 	} else {
// 		// err != nil and too many retries
// 		logger.Logger().Error().Msgf("error processing %s (giving up): %v", newEvent.(InformerEvent).key, err)
// 		w.queue.Forget(newEvent)
// 		utilruntime.HandleError(err)
// 	}
// 	return true
// }

// type triggerType string

// const (
// 	CreateType triggerType = "CREATE"
// 	UpdateType triggerType = "UPDATE"
// 	DeleteType triggerType = "DELETE"
// )

// // TODO: Enhance event creation using client-side caching machanisms - pending
// func (w *watcher) processItem(newEvent InformerEvent) error {
// 	// NOTE that obj will be nil on deletes!
// 	obj, _, err := w.informer.GetIndexer().GetByKey(newEvent.key)

// 	if err != nil {
// 		return fmt.Errorf("error fetching object with key %s from store: %v", newEvent.key, err)
// 	}
// 	// get object's metedata
// 	objectMeta := utils.GetObjectMetaData(obj)

// 	// namespace retrived from event key in case namespace value is empty
// 	if newEvent.namespace == "" && strings.Contains(newEvent.key, "/") {
// 		substring := strings.Split(newEvent.key, "/")
// 		newEvent.namespace = substring[0]
// 		newEvent.key = substring[1]
// 	} else {
// 		newEvent.namespace = objectMeta.Namespace
// 	}

// 	// process events based on its type
// 	var tt triggerType
// 	switch newEvent.eventType {
// 	case "create":
// 		// compare CreationTimestamp and serverStartTime and alert only on latest events
// 		// Could be Replaced by using Delta or DeltaFIFO
// 		if objectMeta.CreationTimestamp.Sub(serverStartTime).Seconds() > 0 {
// 			tt = CreateType
// 		}
// 	case "update":
// 		tt = UpdateType
// 	case "delete":
// 		tt = DeleteType
// 	}
// 	w.events <- Event{
// 		Kind:        newEvent.resourceType,
// 		TriggetType: tt,
// 		Obj:         newEvent.obj,
// 	}
// 	return nil
// }

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	appsv1 "k8s.io/client-go/informers/apps/v1"
	v1 "k8s.io/client-go/informers/core/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type ResourceType string

const (
	SERVICE     = "Service"
	POD         = "Pod"
	REPLICASET  = "ReplicaSet"
	DEPLOYMENT  = "Deployment"
	ENDPOINTS   = "Endpoints"
	CONTAINER   = "Container"
	DAEMONSET   = "DaemonSet"
	STATEFULSET = "StatefulSet"
)

const (
	ADD    = "Add"
	UPDATE = "Update"
	DELETE = "Delete"
)

var k8sVersion string
var resyncPeriod time.Duration = 120 * time.Second

type Collector struct {
	ctx              context.Context
	informersFactory informers.SharedInformerFactory
	watchers         map[ResourceType]cache.SharedIndexInformer
	stopper          chan struct{} // stop signal for the informers
	doneChan         chan struct{} // done signal for k8sCollector
	// watchers
	podInformer         v1.PodInformer
	serviceInformer     v1.ServiceInformer
	replicasetInformer  appsv1.ReplicaSetInformer
	deploymentInformer  appsv1.DeploymentInformer
	endpointsInformer   v1.EndpointsInformer
	daemonsetInformer   appsv1.DaemonSetInformer
	statefulSetInformer appsv1.StatefulSetInformer

	Events chan any
}

func (k *Collector) Start(events chan any) error {
	logger.Logger().Info().Msg("KubernetesCollector initializing...")
	k.Events = events

	// Pod
	k.podInformer = k.informersFactory.Core().V1().Pods()
	k.watchers[POD] = k.podInformer.Informer()

	// Service
	k.serviceInformer = k.informersFactory.Core().V1().Services()
	k.watchers[SERVICE] = k.informersFactory.Core().V1().Services().Informer()

	// ReplicaSet
	k.replicasetInformer = k.informersFactory.Apps().V1().ReplicaSets()
	k.watchers[REPLICASET] = k.replicasetInformer.Informer()

	// Deployment
	k.deploymentInformer = k.informersFactory.Apps().V1().Deployments()
	k.watchers[DEPLOYMENT] = k.deploymentInformer.Informer()

	// Endpoints
	k.endpointsInformer = k.informersFactory.Core().V1().Endpoints()
	k.watchers[ENDPOINTS] = k.endpointsInformer.Informer()

	// DaemonSet
	k.daemonsetInformer = k.informersFactory.Apps().V1().DaemonSets()
	k.watchers[DAEMONSET] = k.daemonsetInformer.Informer()

	// StatefulSet
	k.statefulSetInformer = k.informersFactory.Apps().V1().StatefulSets()
	k.watchers[STATEFULSET] = k.statefulSetInformer.Informer()

	defer runtime.HandleCrash()

	// Add event handlers
	k.watchers[POD].AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    getOnAddPodFunc(k.Events),
		UpdateFunc: getOnUpdatePodFunc(k.Events),
		DeleteFunc: getOnDeletePodFunc(k.Events),
	})

	k.watchers[SERVICE].AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    getOnAddServiceFunc(k.Events),
		UpdateFunc: getOnUpdateServiceFunc(k.Events),
		DeleteFunc: getOnDeleteServiceFunc(k.Events),
	})

	k.watchers[REPLICASET].AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    getOnAddReplicaSetFunc(k.Events),
		UpdateFunc: getOnUpdateReplicaSetFunc(k.Events),
		DeleteFunc: getOnDeleteReplicaSetFunc(k.Events),
	})

	k.watchers[DEPLOYMENT].AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    getOnAddDeploymentSetFunc(k.Events),
		UpdateFunc: getOnUpdateDeploymentSetFunc(k.Events),
		DeleteFunc: getOnDeleteDeploymentSetFunc(k.Events),
	})

	k.watchers[ENDPOINTS].AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    getOnAddEndpointsSetFunc(k.Events),
		UpdateFunc: getOnUpdateEndpointsSetFunc(k.Events),
		DeleteFunc: getOnDeleteEndpointsSetFunc(k.Events),
	})

	k.watchers[DAEMONSET].AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    getOnAddDaemonSetFunc(k.Events),
		UpdateFunc: getOnUpdateDaemonSetFunc(k.Events),
		DeleteFunc: getOnDeleteDaemonSetFunc(k.Events),
	})

	k.watchers[STATEFULSET].AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    getOnAddStatefulSetFunc(k.Events),
		UpdateFunc: getOnUpdateStatefulSetFunc(k.Events),
		DeleteFunc: getOnDeleteStatefulSetFunc(k.Events),
	})

	wg := sync.WaitGroup{}
	wg.Add(len(k.watchers))
	for _, watcher := range k.watchers {
		go func(watcher cache.SharedIndexInformer) {
			watcher.Run(k.stopper) // it will return when stopper is closed
			wg.Done()
		}(watcher)
	}
	wg.Wait()
	logger.Logger().Info().Msg("KuberentesCollector informers stopped")
	k.doneChan <- struct{}{}

	return nil
}

func (k *Collector) Done() <-chan struct{} {
	return k.doneChan
}

func New(c *config.Configuration, parentCtx context.Context) (*Collector, error) {
	ctx, cancel := context.WithCancel(parentCtx)
	// get incluster kubeconfig
	var kubeconfig *string
	var kubeConfig *rest.Config

	if !c.Kubernetes.InCluster {
		var err error
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}

		flag.Parse()

		kubeConfig, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			defer cancel()
			panic(err)
		}
	} else {
		// in cluster config, default
		var err error
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			defer cancel()
			return nil, fmt.Errorf("unable to get incluster kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		defer cancel()
		return nil, fmt.Errorf("unable to create clientset: %w", err)
	}

	version, err := clientset.ServerVersion()
	if err != nil {
		defer cancel()
		return nil, fmt.Errorf("unable to get k8s server version: %w", err)
	}

	k8sVersion = version.String()

	factory := informers.NewSharedInformerFactory(clientset, resyncPeriod)

	collector := &Collector{
		ctx:              ctx,
		stopper:          make(chan struct{}),
		doneChan:         make(chan struct{}),
		informersFactory: factory,
		watchers:         map[ResourceType]cache.SharedIndexInformer{},
	}

	go func(c *Collector) {
		<-c.ctx.Done() // wait for context to be cancelled
		defer cancel()
		c.close()
	}(collector)

	return collector, nil
}

func (k *Collector) GetK8sVersion() string {
	return k8sVersion
}

func (k *Collector) close() {
	logger.Logger().Info().Msg("KubernetesCollector closing...")
	close(k.stopper) // stop informers
}

type NamespaceResources struct {
	Pods     map[string]corev1.Pod     `json:"pods"`     // map[podName]Pod
	Services map[string]corev1.Service `json:"services"` // map[serviceName]Service
}

type ResourceMessage struct {
	ResourceType string `json:"type"`
	EventType    string `json:"eventType"`
	Object       any    `json:"object"`
}
