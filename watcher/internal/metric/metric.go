package metric

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/config"
	"github.com/opisvigilant/futura/watcher/internal/logger"
	"github.com/opisvigilant/futura/watcher/internal/models"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pb "github.com/opisvigilant/futura/proto/events/gen"
)

type Collector struct {
	ctx      context.Context
	doneChan chan struct{} // done signal for metricCollector

	pbc                 pb.CollectServiceClient
	kubernetesInCluster bool
}

func New(cfg *config.Configuration, parentCtx context.Context) (*Collector, error) {
	ctx, cancel := context.WithCancel(parentCtx)

	address := fmt.Sprintf("%s:%s", cfg.Collect.Host, cfg.Collect.Port)
	conn, err := grpc.NewClient(address)
	if err != nil {
		defer cancel()
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	client := pb.NewCollectServiceClient(conn)

	collector := &Collector{
		ctx:                 ctx,
		doneChan:            make(chan struct{}),
		pbc:                 client,
		kubernetesInCluster: cfg.Kubernetes.InCluster,
	}

	go func(c *Collector) {
		<-c.ctx.Done() // wait for context to be cancelled
		defer cancel()
		c.close()
	}(collector)

	return collector, nil
}

func (c *Collector) Start(interval time.Duration, excludedNamespaces []string) error {
	// get incluster kubeconfig
	var kubeconfig *string
	var kubeConfig *rest.Config

	if !c.kubernetesInCluster {
		var err error
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}

		flag.Parse()

		kubeConfig, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return err
		}
	} else {
		// in cluster config, default
		var err error
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return fmt.Errorf("unable to get incluster kubeconfig: %w", err)
		}
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return fmt.Errorf("unable to create kubeClient: %w", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Convert exclude list to map for fast lookup
	excludeMap := make(map[string]bool)
	for _, ns := range excludedNamespaces {
		excludeMap[strings.TrimSpace(ns)] = true
	}

	ctx := context.Background()

	for {
		select {
		case <-ticker.C:
			hostname, err := os.Hostname()
			if err != nil {
				return err
			}

			summary, err := fetchSummary(hostname)
			if err != nil {
				log.Printf("summary error: %v", err)
				continue
			}

			var batch []*pb.ContainerMetric

			for _, pod := range summary.Pods {
				ns := pod.PodRef.Namespace
				name := pod.PodRef.Name
				for _, c := range pod.Containers {
					cpu := float64(c.CPU.UsageNanoCores) / 1e9
					mem := c.Memory.UsageBytes
					memWS := c.Memory.WorkingSetBytes
					fs := c.Rootfs.UsedBytes
					rx := c.Network.RxBytes
					tx := c.Network.TxBytes

					// PodSpec limits
					cpuLimit, memLimit := getLimitsForContainer(kubeClient, ns, name, c.Name)

					batch = append(batch, &pb.ContainerMetric{
						Metadata: &pb.MetricMetadata{
							ClusterId:     "my-cluster",
							NodeName:      hostname,
							Namespace:     ns,
							PodName:       name,
							ContainerName: c.Name,
							Source:        "kubelet",
							TimestampUtc:  time.Now().UTC().Format(time.RFC3339),
						},
						CpuUsageCores:         cpu,
						MemoryUsageBytes:      mem,
						MemoryWorkingSetBytes: memWS,
						RxBytes:               rx,
						TxBytes:               tx,
						FsUsageBytes:          fs,
						CpuLimitCores:         cpuLimit,
						MemoryLimitBytes:      memLimit,
					})
				}
			}

			_, err = c.pbc.SendMetric(ctx, &pb.ContainerMetricBatch{
				Metrics: batch,
			})
			if err != nil {
				return err
			}
			logger.Logger().Info().Msgf("✅ Sent %d metrics", len(batch))
		case <-ctx.Done():
			logger.Logger().Info().Msg("Shutting down metrics scraper...")
			return nil
		}
	}
}

func readToken() (string, error) {
	b, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", fmt.Errorf("failed to read token: %v", err)
	}
	return strings.TrimSpace(string(b)), nil
}

func fetchSummary(kubeletHost string) (*models.MetricSummary, error) {
	url := fmt.Sprintf("https://%s:10250/stats/summary", kubeletHost)
	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	req, _ := http.NewRequest("GET", url, nil)
	token, err := readToken()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var summary models.MetricSummary
	err = json.Unmarshal(body, &summary)
	return &summary, err
}

func getLimitsForContainer(client *kubernetes.Clientset, ns, podName, containerName string) (float64, uint64) {
	pod, err := client.CoreV1().Pods(ns).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		log.Printf("cannot get pod %s/%s: %v", ns, podName, err)
		return 0, 0
	}

	for _, c := range pod.Spec.Containers {
		if c.Name == containerName {
			cpu := float64(c.Resources.Limits.Cpu().MilliValue()) / 1000.0
			mem := uint64(c.Resources.Limits.Memory().Value())
			return cpu, mem
		}
	}
	return 0, 0
}

func (c *Collector) Done() <-chan struct{} {
	return c.doneChan
}

func (c *Collector) close() {
	logger.Logger().Info().Msg("MetricCollector closing...")
}
