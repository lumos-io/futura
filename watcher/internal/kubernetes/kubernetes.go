package kubernetes

import (
	"context"
	"sync"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	k8s "github.com/opisvigilant/futura/watcher/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	appsv1 "k8s.io/client-go/informers/apps/v1"
	v1 "k8s.io/client-go/informers/core/v1"

	"k8s.io/client-go/tools/cache"
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

func New(k8sClient *k8s.Client, c *config.Configuration, parentCtx context.Context) (*Collector, error) {
	ctx, cancel := context.WithCancel(parentCtx)

	var err error
	k8sVersion, err = k8sClient.GetVersion()
	if err != nil {
		defer cancel()
		return nil, err
	}

	factory := informers.NewSharedInformerFactory(k8sClient.RawClient(), resyncPeriod)

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
