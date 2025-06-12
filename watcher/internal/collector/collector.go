package collector

// aggregate data from different sources
// 1. k8s
// 2. metrics-server (TODO)

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/opisvigilant/futura/watcher/internal/handlers"
	"github.com/opisvigilant/futura/watcher/internal/kubernetes"
	"github.com/opisvigilant/futura/watcher/internal/logger"
)

var maxPid int

func init() {
	var err error
	maxPid, err = getPidMax()
	if err != nil {
		logger.Logger().Fatal().Err(err).Msg("error getting max pid")
	}
}

type Collector struct {
	ctx context.Context

	stopper  chan struct{} // stop signal for the informers
	doneChan chan struct{} // done signal for k8sCollector

	// store the service map
	clusterInfo *ClusterInfo

	// send data to datastore
	eventsHandler handlers.Handler
}

func NewCollector(parentCtx context.Context, eventHandler handlers.Handler) *Collector {
	ctx, _ := context.WithCancel(parentCtx)

	collector := &Collector{
		ctx:           ctx,
		doneChan:      make(chan struct{}),
		eventsHandler: eventHandler,
	}

	collector.clusterInfo = newClusterInfo(liveProcCount)

	go func(c *Collector) {
		<-c.ctx.Done() // wait for context to be cancelled
		c.close()
	}(collector)

	return collector
}

func (c *Collector) Run(k8sChan <-chan any) {
	go c.processk8s(k8sChan)

	//TODO: progress metrics-server signal here
	// ...
}

func (c *Collector) processk8s(k8sChan <-chan any) {
	for data := range k8sChan {
		d := data.(kubernetes.ResourceMessage)
		switch d.ResourceType {
		case kubernetes.POD:
			c.processPod(d)
		case kubernetes.SERVICE:
			c.processSvc(d)
		case kubernetes.REPLICASET:
			c.processReplicaSet(d)
		case kubernetes.DEPLOYMENT:
			c.processDeployment(d)
		case kubernetes.ENDPOINTS:
			c.processEndpoints(d)
		case kubernetes.CONTAINER:
			c.processContainer(d)
		case kubernetes.DAEMONSET:
			c.processDaemonSet(d)
		case kubernetes.STATEFULSET:
			c.processStatefulSet(d)
		default:
			logger.Logger().Warn().Msgf("unknown resource type %s", d.ResourceType)
		}
	}

	c.eventsHandler.HandleKubernetesEvent()
}

func (c *Collector) Done() <-chan struct{} {
	return c.doneChan
}

func (c *Collector) close() {
	logger.Logger().Info().Msg("Collector closing...")
}

func getPidMax() (int, error) {
	// Read the contents of the file
	f, err := os.Open("/proc/sys/kernel/pid_max")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return 0, err
	}
	content, err := io.ReadAll(f)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return 0, err
	}

	// Convert the content to an integer
	pidMax, err := strconv.Atoi(string(content[:len(content)-1])) // trim newline
	if err != nil {
		fmt.Println("Error converting to integer:", err)
		return 0, err
	}
	return pidMax, nil
}
