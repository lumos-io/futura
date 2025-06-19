package sender

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/internal/models"

	pb "github.com/opisvigilant/futura/proto/events/gen"
)

// Sender handler implements handler.Handler interface,
// Notify event to Sender
type Sender struct {
	ctx       context.Context
	pbc       pb.CollectServiceClient
	batchSize int

	PodEventChan         chan any // *PodEvent
	ServiceEventChan     chan any // *SvcEvent
	DeploymentEventChan  chan any // *DepEvent
	ReplicaSetEventChan  chan any // *RsEvent
	EndpointEventChan    chan any // *EndpointsEvent
	ContainerEventChan   chan any // *ContainerEvent
	DaemonSetEventChan   chan any // *DaemonSetEvent
	StatefulSetEventChan chan any // *StatefulSetEvent
	JobEventChan         chan any // *JobEvent
	CronJobEventChan     chan any // *CronJobEvent
}

var tag string
var kernelVersion string
var cloudProvider CloudProvider

func extractKernelVersion() string {
	// Path to the /proc/version file
	filePath := "/proc/version"
	file, err := os.Open(filePath)
	if err != nil {
		logger.Logger().Fatal().AnErr("error", err).Msgf("Unable to open file %s", filePath)
	}

	// Read the content of the file
	content, err := io.ReadAll(file)
	if err != nil {
		logger.Logger().Fatal().AnErr("error", err).Msgf("Unable to read file %s", filePath)
	}

	// Convert the content to a string
	versionInfo := string(content)

	// Split the versionInfo string into lines
	lines := strings.Split(versionInfo, "\n")

	// Extract the kernel version from the first line
	// Assuming the kernel version is the first word in the first line
	if len(lines) > 0 {
		fields := strings.Fields(lines[0])
		if len(fields) > 2 {
			return fields[2]
		}
	}
	return "Unable to extract kernel version"
}

type CloudProvider string

const (
	CloudProviderAWS          CloudProvider = "AWS"
	CloudProviderGCP          CloudProvider = "GCP"
	CloudProviderAzure        CloudProvider = "Azure"
	CloudProviderDigitalOcean CloudProvider = "DigitalOcean"
	CloudProviderUnknown      CloudProvider = "Unknown"
)

func getCloudProvider() CloudProvider {
	if vendor, err := os.ReadFile("/sys/class/dmi/id/board_vendor"); err == nil {
		switch strings.TrimSpace(string(vendor)) {
		case "Amazon EC2":
			return CloudProviderAWS
		case "Google":
			return CloudProviderGCP
		case "Microsoft Corporation":
			return CloudProviderAzure
		case "DigitalOcean":
			return CloudProviderDigitalOcean
		}
	}
	return CloudProviderUnknown
}

// Init prepares Webhook configuration
func New(c *config.Configuration) (*Sender, error) {
	tag = c.Tag

	logger.Logger().Info().Str("tag", tag).Msg("watcher tag")

	// kernelVersion = extractKernelVersion()
	// cloudProvider = getCloudProvider()

	address := fmt.Sprintf("%s:%s", c.Collect.Host, c.Collect.Port)
	conn, err := grpc.NewClient(address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	client := pb.NewCollectServiceClient(conn)

	resourceChanSize := 200
	s := &Sender{
		ctx:                  context.TODO(),
		batchSize:            1000,
		pbc:                  client,
		PodEventChan:         make(chan any, 5*resourceChanSize),
		ServiceEventChan:     make(chan any, 2*resourceChanSize),
		ReplicaSetEventChan:  make(chan any, 2*resourceChanSize),
		DeploymentEventChan:  make(chan any, 2*resourceChanSize),
		EndpointEventChan:    make(chan any, resourceChanSize),
		ContainerEventChan:   make(chan any, 5*resourceChanSize),
		DaemonSetEventChan:   make(chan any, resourceChanSize),
		StatefulSetEventChan: make(chan any, resourceChanSize),
		JobEventChan:         make(chan any, 2*resourceChanSize),
		CronJobEventChan:     make(chan any, 2*resourceChanSize),
	}

	// events are resynced every 60 seconds on kubernetes informers
	// resourceBatchSize ~ burst size, if more than resourceBatchSize events are sent in a moment, blocking can occur
	// resync period / event interval = 60 / 5 = 12
	// 12 * resourceBatchSize = 12 * 1000 = 12000
	// it can send upto 12k events in 60 seconds
	// seems safe enough, if not, we can increase the buffer size
	eventsInterval := 5 * time.Second
	go s.sendEventsInBatch(s.PodEventChan, eventsInterval)
	go s.sendEventsInBatch(s.ServiceEventChan, eventsInterval)
	go s.sendEventsInBatch(s.ReplicaSetEventChan, eventsInterval)
	go s.sendEventsInBatch(s.DeploymentEventChan, eventsInterval)
	go s.sendEventsInBatch(s.EndpointEventChan, eventsInterval)
	go s.sendEventsInBatch(s.ContainerEventChan, eventsInterval)
	go s.sendEventsInBatch(s.DaemonSetEventChan, eventsInterval)
	go s.sendEventsInBatch(s.StatefulSetEventChan, eventsInterval)

	return s, nil
}

var resourceBatchSize int64 = 50

func (b *Sender) sendEventsInBatch(ch chan any, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-b.ctx.Done():
			logger.Logger().Info().Msg("stopping sending events to backend")
			return
		case <-t.C:
			randomDuration := time.Duration(rand.Intn(50)) * time.Millisecond
			time.Sleep(randomDuration)

			b.send(ch)
		}
	}
}

func (b *Sender) send(ch <-chan any) {
	batch := make([]any, 0, resourceBatchSize)
	loop := true

	for i := 0; (i < int(resourceBatchSize)) && loop; i++ {
		select {
		case ev := <-ch:
			batch = append(batch, ev)
		case <-time.After(100 * time.Millisecond):
			loop = false
		}
	}

	if len(batch) == 0 {
		return
	}
	
	// payload := models.EventPayload{
	// 	Metadata: models.Metadata{
	// 		IdempotencyKey: uuid.NewString(),
	// 		WatcherVersion: tag,
	// 	},
	// 	Events: batch,
	// }

	payload := &pb.KubernetesEventBatch{
		
	}


}


type HealthCheckAction string

const (
	HealthCheckActionStop HealthCheckAction = "payment_required"
	HealthCheckActionOK   HealthCheckAction = "ok"
)

func (b *Sender) SendHealthCheck(tracing bool, metrics bool, logs bool, nsFilter string, k8sVersion string) chan HealthCheckAction {
	// t := time.NewTicker(10 * time.Second)
	// defer t.Stop()

	ch := make(chan HealthCheckAction)

	// createHealthCheckPayload := func() models.HealthCheckPayload {
	// 	return models.HealthCheckPayload{
	// 		Metadata: models.Metadata{
	// 			IdempotencyKey: uuid.NewString(),
	// 			WatcherVersion: tag,
	// 		},
	// 		Telemetry: struct {
	// 			KernelVersion string `json:"kernel_version"`
	// 			K8sVersion    string `json:"k8s_version"`
	// 			CloudProvider string `json:"cloud_provider"`
	// 		}{
	// 			KernelVersion: kernelVersion,
	// 			K8sVersion:    k8sVersion,
	// 			CloudProvider: string(cloudProvider),
	// 		},
	// 	}
	// }

	// f := func() {
	// 	payloadBytes, err := json.Marshal(createHealthCheckPayload())
	// 	if err != nil {
	// 		logger.Logger().Error().Msgf("error marshalling batch: %v", err)
	// 		return
	// 	}

	// 	req, err := http.NewRequest(http.MethodPut, b.URL, bytes.NewBuffer(payloadBytes))
	// 	if err != nil {
	// 		logger.Logger().Error().Msgf("error creating http request: %v", err)
	// 		return
	// 	}

	// 	req.Header.Set("Content-Type", "application/json")
	// 	req.Header.Set("Accept", "application/json")

	// 	if err := b.DoRequest(req); err != nil {
	// 		logger.Logger().Error().Msgf("error sending healtcheck request, %v", err)
	// 		return
	// 	}
	// }

	// go func() {
	// 	for range t.C {
	// 		f()
	// 	}
	// }()

	return ch
}

// func (b *Sender) scrapeNodeMetrics() (io.Reader, error) {
// 	// get node metrics from node-exporter
// 	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/inner/metrics", innerMetricsPort), nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("error creating inner metrics request: %v", err)
// 	}

// 	ctx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
// 	// defer cancel()
// 	// do not defer cancel here, since we return the reader to the caller on success
// 	// if deferred, there will be a race condition between the caller and the defer

// 	// use the default client, ds client reads response on success to look for failed events,
// 	// therefore body here will be empty
// 	resp, err := http.DefaultClient.Do(req.WithContext(ctx))

// 	if err != nil {
// 		cancel()
// 		return nil, fmt.Errorf("error sending inner metrics request: %v", err)
// 	}

// 	if resp.StatusCode != http.StatusOK {
// 		cancel()
// 		return nil, fmt.Errorf("inner metrics request not success: %d", resp.StatusCode)
// 	}

// 	return resp.Body, nil
// }
