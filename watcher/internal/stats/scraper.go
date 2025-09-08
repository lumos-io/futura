package stats

import (
	"errors"
	"fmt"
	"os"
	"sync"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/stats/kubelet"
	"github.com/opisvigilant/futura/watcher/pkg/kubernetes"
	"github.com/rs/zerolog/log"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type KubeletScraper struct {
	restClient       kubelet.RestClient
	statsProvider    *kubelet.StatsProvider
	metadataProvider *kubelet.MetadataProvider

	k8sClient    k8s.Interface
	nodeInformer cache.SharedInformer
	stopCh       chan struct{}
	m            sync.RWMutex

	// A struct that keeps Node's information
	nodeInfo *kubelet.NodeInfo
}

func NewKubeletScraper(config *config.Configuration, k8sClient k8s.Interface) (*KubeletScraper, error) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return nil, errors.New("NODE_NAME env variable is not defined")
	}

	// FIXME: this is only on the read-only port
	// I need to handle the HTTPS and CA/CertFile situation
	endpoint := fmt.Sprintf("https://%s:10250", nodeName)
	clientProvider, err := kubernetes.NewClientProvider(endpoint, config)
	if err != nil {
		return nil, err
	}
	client, err := clientProvider.BuildClient()
	if err != nil {
		return nil, err
	}
	rest := kubelet.NewRestClient(client)

	return &KubeletScraper{
		restClient:       rest,
		statsProvider:    kubelet.NewStatsProvider(rest),
		metadataProvider: kubelet.NewMetadataProvider(rest),
		k8sClient:        k8sClient,
		stopCh:           make(chan struct{}),
		nodeInfo:         &kubelet.NodeInfo{},
	}, nil
}

func (ks *KubeletScraper) DoScrape() (*pb.KubernetesKubeletStats, error) {
	summary, err := ks.statsProvider.StatsSummary()
	if err != nil {
		log.Logger.Error().Err(err).Msg("call to /stats/summary endpoint failed")
		return nil, err
	}

	podsMetadata, err := ks.metadataProvider.Pods()
	if err != nil {
		log.Logger.Error().Err(err).Msg("call to /pods endpoint failed")
		return nil, err
	}

	var nodeInfo kubelet.NodeInfo
	if ks.nodeInformer != nil {
		nodeInfo = ks.node()
	}

	metaD := kubelet.NewMetadata(podsMetadata, nodeInfo)
	accumulator := kubelet.MetricsData(summary, metaD)

	return accumulator.Emit(), nil
}

func (ks *KubeletScraper) Init() error {
	if ks.nodeInformer != nil {
		_, err := ks.nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    ks.handleNodeAdd,
			UpdateFunc: ks.handleNodeUpdate,
		})
		if err != nil {
			log.Logger.Error().Err(err).Msg("error adding event handler to node informer")
			return err
		}
		go ks.nodeInformer.Run(ks.stopCh)
	}
	return nil
}

func (ks *KubeletScraper) Shutdown() error {
	log.Logger.Debug().Msg("KubeStatsCollector executing close...")
	if ks.stopCh != nil {
		close(ks.stopCh)
	}
	return nil
}

func (ks *KubeletScraper) handleNodeAdd(obj any) {
	if node, ok := obj.(*v1.Node); ok {
		ks.addOrUpdateNode(node)
	} else {
		log.Logger.Error().Interface("received", obj).Msg("object received was not of type v1.Node")
	}
}

func (ks *KubeletScraper) handleNodeUpdate(_, newNode any) {
	if node, ok := newNode.(*v1.Node); ok {
		ks.addOrUpdateNode(node)
	} else {
		log.Logger.Error().Interface("received", newNode).Msg("object received was not of type v1.Node")
	}
}

func (ks *KubeletScraper) addOrUpdateNode(node *v1.Node) {
	ks.m.Lock()
	defer ks.m.Unlock()

	if cpu, ok := node.Status.Capacity["cpu"]; ok {
		if q, err := resource.ParseQuantity(cpu.String()); err == nil {
			ks.nodeInfo.CPUCapacity = float64(q.MilliValue()) / 1000
		}
	}
	if memory, ok := node.Status.Capacity["memory"]; ok {
		// ie: 32564740Ki
		if q, err := resource.ParseQuantity(memory.String()); err == nil {
			ks.nodeInfo.MemoryCapacity = float64(q.Value())
		}
	}
}

func (ks *KubeletScraper) node() kubelet.NodeInfo {
	ks.m.RLock()
	defer ks.m.RUnlock()
	return *ks.nodeInfo
}
