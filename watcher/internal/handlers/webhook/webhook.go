package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/internal/models"
)

// Webhook handler implements handler.Handler interface,
// Notify event to Webhook
type Webhook struct {
	URL string

	ctx       context.Context
	hc        *http.Client
	batchSize uint64

	podEventChan       chan any // *PodEvent
	svcEventChan       chan any // *SvcEvent
	depEventChan       chan any // *DepEvent
	rsEventChan        chan any // *RsEvent
	epEventChan        chan any // *EndpointsEvent
	containerEventChan chan any // *ContainerEvent
	dsEventChan        chan any // *DaemonSetEvent
	ssEventChan        chan any // *StatefulSetEvent
	jobEventChan       chan any // *JobEvent
	cronJobEventChan   chan any // *CronJobEvent
}

const (
	podEndpoint         = "/pod/"
	svcEndpoint         = "/svc/"
	rsEndpoint          = "/replicaset/"
	depEndpoint         = "/deployment/"
	epEndpoint          = "/endpoint/"
	containerEndpoint   = "/container/"
	dsEndpoint          = "/daemonset/"
	ssEndpoint          = "/statefulset/"
	jobEndpoint         = "/job/"
	cronJobEndpoint     = "/cronjob/"
	healthCheckEndpoint = "/healthcheck/"
)

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
func (w *Webhook) Init(c *config.Configuration) error {
	tag = c.Tag
	batchSize := c.Handler.Webhook.BatchSize

	logger.Logger().Info().Str("tag", tag).Msg("watcher tag")

	// kernelVersion = extractKernelVersion()
	// cloudProvider = getCloudProvider()

	w.URL = c.Handler.Webhook.URL
	w.ctx = context.TODO()
	w.hc = http.DefaultClient
	w.batchSize = batchSize

	resourceChanSize := 200
	w.podEventChan = make(chan any, 5*resourceChanSize)
	w.svcEventChan = make(chan any, 2*resourceChanSize)
	w.rsEventChan = make(chan any, 2*resourceChanSize)
	w.depEventChan = make(chan any, 2*resourceChanSize)
	w.epEventChan = make(chan any, resourceChanSize)
	w.containerEventChan = make(chan any, 5*resourceChanSize)
	w.dsEventChan = make(chan any, resourceChanSize)
	w.ssEventChan = make(chan any, resourceChanSize)
	w.jobEventChan = make(chan any, 2*resourceChanSize)
	w.cronJobEventChan = make(chan any, 2*resourceChanSize)

	return nil
}

func (w *Webhook) HandleKubernetesEvent() {
	// events are resynced every 60 seconds on kubernetes informers
	// resourceBatchSize ~ burst size, if more than resourceBatchSize events are sent in a moment, blocking can occur
	// resync period / event interval = 60 / 5 = 12
	// 12 * resourceBatchSize = 12 * 1000 = 12000
	// it can send upto 12k events in 60 seconds
	// seems safe enough, if not, we can increase the buffer size
	eventsInterval := 5 * time.Second
	go w.sendEventsInBatch(w.podEventChan, podEndpoint, eventsInterval)
	go w.sendEventsInBatch(w.svcEventChan, svcEndpoint, eventsInterval)
	go w.sendEventsInBatch(w.rsEventChan, rsEndpoint, eventsInterval)
	go w.sendEventsInBatch(w.depEventChan, depEndpoint, eventsInterval)
	go w.sendEventsInBatch(w.epEventChan, epEndpoint, eventsInterval)
	go w.sendEventsInBatch(w.containerEventChan, containerEndpoint, eventsInterval)
	go w.sendEventsInBatch(w.dsEventChan, dsEndpoint, eventsInterval)
	go w.sendEventsInBatch(w.ssEventChan, ssEndpoint, eventsInterval)

}

var resourceBatchSize int64 = 50

func (w *Webhook) DoRequest(req *http.Request) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()

	resp, err := w.hc.Do(req.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("error sending http request: %v", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body) // in order to reuse the connection
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("req failed: %d, %s", resp.StatusCode, string(body))
	}

	return nil
}

func (b *Webhook) sendEventsInBatch(ch chan any, endpoint string, interval time.Duration) {
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

			b.send(ch, endpoint)
		}
	}
}

func (b *Webhook) send(ch <-chan any, endpoint string) {
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

	payload := models.EventPayload{
		Metadata: models.Metadata{
			IdempotencyKey: string(uuid.NewUUID()),
			WatcherVersion: tag,
		},
		Events: batch,
	}

	b.sendToBackend(http.MethodPost, payload, endpoint)
}

func (w *Webhook) sendToBackend(method string, payload interface{}, endpoint string) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Logger().Error().Msgf("error marshalling batch: %v", err)
		return
	}

	httpReq, err := http.NewRequest(method, w.URL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.Logger().Error().Msgf("error creating http request: %v", err)
		return
	}

	// if endpoint == reqEndpoint {
	// 	logger.Logger().Debug().Str("endpoint", endpoint).Any("payload", payload).Msg("sending batch to backend")
	// }
	err = w.DoRequest(httpReq)
	if err != nil {
		logger.Logger().Error().Msgf("backend persist error at ep %s : %v", endpoint, err)
	}
}

// ------------------------------

func (b *Webhook) PersistPod(pod models.Pod, eventType string) error {
	podEvent := models.ConvertPodToPodEvent(pod, eventType)
	b.podEventChan <- &podEvent
	return nil
}

func (b *Webhook) PersistService(service models.Service, eventType string) error {
	svcEvent := models.ConvertSvcToSvcEvent(service, eventType)
	b.svcEventChan <- &svcEvent
	return nil
}

func (b *Webhook) PersistDeployment(d models.Deployment, eventType string) error {
	depEvent := models.ConvertDepToDepEvent(d, eventType)
	b.depEventChan <- &depEvent
	return nil
}

func (b *Webhook) PersistReplicaSet(rs models.ReplicaSet, eventType string) error {
	rsEvent := models.ConvertRsToRsEvent(rs, eventType)
	b.rsEventChan <- &rsEvent
	return nil
}

func (b *Webhook) PersistEndpoints(ep models.Endpoints, eventType string) error {
	epEvent := models.ConvertEpToEpEvent(ep, eventType)
	b.epEventChan <- &epEvent
	return nil
}

func (b *Webhook) PersistDaemonSet(ds models.DaemonSet, eventType string) error {
	dsEvent := models.ConvertDsToDsEvent(ds, eventType)
	b.dsEventChan <- &dsEvent
	return nil
}

func (b *Webhook) PersistStatefulSet(ss models.StatefulSet, eventType string) error {
	ssEvent := models.ConvertSsToSsEvent(ss, eventType)
	b.ssEventChan <- &ssEvent
	return nil
}

func (b *Webhook) PersistContainer(c models.Container, eventType string) error {
	cEvent := models.ConvertContainerToContainerEvent(c, eventType)
	b.containerEventChan <- &cEvent
	return nil
}

type HealthCheckAction string

const (
	HealthCheckActionStop HealthCheckAction = "payment_required"
	HealthCheckActionOK   HealthCheckAction = "ok"
)

func (b *Webhook) SendHealthCheck(tracing bool, metrics bool, logs bool, nsFilter string, k8sVersion string) chan HealthCheckAction {
	t := time.NewTicker(10 * time.Second)
	// defer t.Stop()

	ch := make(chan HealthCheckAction)

	createHealthCheckPayload := func() models.HealthCheckPayload {
		return models.HealthCheckPayload{
			Metadata: models.Metadata{
				IdempotencyKey: uuid.NewString(),
				WatcherVersion: tag,
			},
			Telemetry: struct {
				KernelVersion string `json:"kernel_version"`
				K8sVersion    string `json:"k8s_version"`
				CloudProvider string `json:"cloud_provider"`
			}{
				KernelVersion: kernelVersion,
				K8sVersion:    k8sVersion,
				CloudProvider: string(cloudProvider),
			},
		}
	}

	f := func() {
		payloadBytes, err := json.Marshal(createHealthCheckPayload())
		if err != nil {
			logger.Logger().Error().Msgf("error marshalling batch: %v", err)
			return
		}

		req, err := http.NewRequest(http.MethodPut, b.URL, bytes.NewBuffer(payloadBytes))
		if err != nil {
			logger.Logger().Error().Msgf("error creating http request: %v", err)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		// defer cancel()

		resp, err := b.c.Do(req.WithContext(ctx))
		if err != nil {
			logger.Logger().Error().Msgf("error sending healtcheck request, %v", err)
			return
		}

		if resp.StatusCode == http.StatusPaymentRequired {
			ch <- HealthCheckActionStop
		} else if resp.StatusCode == http.StatusOK {
			ch <- HealthCheckActionOK
		}

		_, _ = io.Copy(io.Discard, resp.Body) // in order to reuse the connection
		resp.Body.Close()
	}

	go func() {
		for range t.C {
			f()
		}
	}()

	return ch
}

func (b *Webhook) scrapeNodeMetrics() (io.Reader, error) {
	// get node metrics from node-exporter
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/inner/metrics", innerMetricsPort), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating inner metrics request: %v", err)
	}

	ctx, cancel := context.WithTimeout(b.ctx, 5*time.Second)
	// defer cancel()
	// do not defer cancel here, since we return the reader to the caller on success
	// if deferred, there will be a race condition between the caller and the defer

	// use the default client, ds client reads response on success to look for failed events,
	// therefore body here will be empty
	resp, err := http.DefaultClient.Do(req.WithContext(ctx))

	if err != nil {
		cancel()
		return nil, fmt.Errorf("error sending inner metrics request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		cancel()
		return nil, fmt.Errorf("inner metrics request not success: %d", resp.StatusCode)
	}

	return resp.Body, nil
}
