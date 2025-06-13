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

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/internal/models"

	"k8s.io/apimachinery/pkg/util/uuid"
)

// Webhook handler implements handler.Handler interface,
// Notify event to Webhook
type Webhook struct {
	URL string

	ctx       context.Context
	hc        *http.Client
	batchSize uint64
}

// set from ldflags
var NodeID string
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
	NodeID = c.NodeName
	tag = c.Tag
	batchSize := c.Handler.Webhook.BatchSize

	logger.Logger().Info().Str("tag", tag).Msg("watcher tag")

	// kernelVersion = extractKernelVersion()
	// cloudProvider = getCloudProvider()

	w.URL = c.Handler.Webhook.URL
	w.ctx = context.TODO()
	w.hc = http.DefaultClient
	w.batchSize = batchSize

	// TODO: remove this below?
	// go w.sendReqsInBatch(uint64(batchSize))

	return nil
}

func (w *Webhook) HandleKubernetesEvent(k8sChan <-chan any) {
	eventsInterval := 10 * time.Second
	go w.sendEventsInBatch(k8sChan, eventsInterval)

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

func (b *Webhook) sendReqsInBatch(batchSize uint64) {
	// t := time.NewTicker(5 * time.Second)
	// defer t.Stop()

	// send := func() {
	// 	batch := make([]*models.ReqInfo, 0, batchSize)
	// 	loop := true

	// 	for i := 0; (i < int(batchSize)) && loop; i++ {
	// 		select {
	// 		case req := <-b.reqChanBuffer:
	// 			batch = append(batch, req)
	// 		case <-time.After(50 * time.Millisecond):
	// 			loop = false
	// 		}
	// 	}

	// 	if len(batch) == 0 {
	// 		return
	// 	}

	// 	reqsPayload := convertReqsToPayload(batch)
	// 	go b.sendToBackend(reqsPayload)

	// 	// return reqInfoss to the pool
	// 	for _, req := range batch {
	// 		b.reqInfoPool.Put(req)
	// 	}
	// }

	// for {
	// 	select {
	// 	case <-b.ctx.Done():
	// 		logger.Logger().Info().Msg("stopping sending reqs to backend")
	// 		return
	// 	case <-t.C:
	// 		send()
	// 	}
	// }
}

func (b *Webhook) send(ch <-chan any) {
	batch := make([]any, 0, resourceBatchSize)
	loop := true

	for i := 0; (i < int(resourceBatchSize)) && loop; i++ {
		select {
		case ev := <-ch:
			batch = append(batch, ev)
		case <-time.After(1 * time.Second):
			loop = false
		}
	}

	if len(batch) == 0 {
		return
	}

	payload := models.EventPayload{
		Metadata: models.Metadata{
			IdempotencyKey: string(uuid.NewUUID()),
			NodeID:         NodeID,
			WatcherVersion: tag,
		},
		Events: batch,
	}
	b.sendToBackend(payload)
}

func (w *Webhook) sendToBackend(payload any) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		logger.Logger().Error().Msgf("error marshalling batch: %v", err)
		return
	}

	httpReq, err := http.NewRequest(http.MethodPost, w.URL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.Logger().Error().Msgf("error creating http request: %v", err)
		return
	}

	err = w.DoRequest(httpReq)
	if err != nil {
		logger.Logger().Error().Msgf("backend persist error: %v", err)
	}
}

func (b *Webhook) sendEventsInBatch(ch <-chan any, interval time.Duration) {
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

/*
func (b *Webhook) PersistPod(pod Pod, eventType string) error {
	podEvent := convertPodToPodEvent(pod, eventType)
	b.podEventChan <- &podEvent
	return nil
}

func (b *Webhook) PersistService(service Service, eventType string) error {
	svcEvent := convertSvcToSvcEvent(service, eventType)
	b.svcEventChan <- &svcEvent
	return nil
}

func (b *Webhook) PersistDeployment(d Deployment, eventType string) error {
	depEvent := convertDepToDepEvent(d, eventType)
	b.depEventChan <- &depEvent
	return nil
}

func (b *Webhook) PersistReplicaSet(rs ReplicaSet, eventType string) error {
	rsEvent := convertRsToRsEvent(rs, eventType)
	b.rsEventChan <- &rsEvent
	return nil
}

func (b *Webhook) PersistEndpoints(ep Endpoints, eventType string) error {
	epEvent := convertEpToEpEvent(ep, eventType)
	b.epEventChan <- &epEvent
	return nil
}

func (b *Webhook) PersistDaemonSet(ds DaemonSet, eventType string) error {
	dsEvent := convertDsToDsEvent(ds, eventType)
	b.dsEventChan <- &dsEvent
	return nil
}

func (b *Webhook) PersistContainer(c Container, eventType string) error {
	cEvent := convertContainerToContainerEvent(c, eventType)
	b.containerEventChan <- &cEvent
	return nil
}
*/

func (b *Webhook) SendHealthCheck(ebpf bool, metrics bool, k8sVersion string) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()

	createHealthCheckPayload := func() models.HealthCheckPayload {
		return models.HealthCheckPayload{
			Info: struct {
				MetricsEnabled bool `json:"metrics"`
			}{
				MetricsEnabled: metrics,
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

	for {
		select {
		case <-b.ctx.Done():
			logger.Logger().Info().Msg("stopping sending health check")
			return
		case <-t.C:
			// TODO: fix the URL for healthcheck
			b.sendToBackend(createHealthCheckPayload())
		}
	}
}
