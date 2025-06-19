package metric

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/models"
	"k8s.io/client-go/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

func RunMetricsScraper(ctx context.Context, interval time.Duration, excludedNamespaces []string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	metricsClient := metricsclient.NewForConfigOrDie(config)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Convert exclude list to map for fast lookup
	excludeMap := make(map[string]bool)
	for _, ns := range excludedNamespaces {
		excludeMap[strings.TrimSpace(ns)] = true
	}

	for {
		select {
		case <-ticker.C:
			podMetricsList, err := metricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
			if err != nil {
				log.Println("Error fetching pod metrics:", err)
				continue
			}

			var metricsPayload []models.PodMetric
			for _, item := range podMetricsList.Items {
				if excludeMap[item.Namespace] {
					continue
				}

				for _, container := range item.Containers {
					metricsPayload = append(metricsPayload, models.PodMetric{
						Timestamp:     item.Timestamp.Time.Format(time.RFC3339),
						Namespace:     item.Namespace,
						Pod:           item.Name,
						Container:     container.Name,
						CPU_millicore: container.Usage.Cpu().MilliValue(),
						Memory_bytes:  container.Usage.Memory().Value(),
					})
				}
			}

			jsonData, err := json.Marshal(metricsPayload)
			if err != nil {
				log.Println("Failed to marshal metrics:", err)
				continue
			}

			fmt.Println(jsonData)

			// resp, err := http.Post(externalAPI, "application/json", bytes.NewBuffer(jsonData))
			// if err != nil {
			// 	log.Println("Failed to send metrics to external API:", err)
			// 	continue
			// }
			// resp.Body.Close()
			// log.Printf("Pushed %d metrics to %s\n", len(metricsPayload), externalAPI)

		case <-ctx.Done():
			log.Println("Shutting down metrics scraper...")
			return nil
		}
	}
}
