package metric

// type Collector struct {
// 	ctx      context.Context
// 	doneChan chan struct{} // done signal for metricCollector

// 	pbc       pb.CollectServiceClient
// 	k8sClient *k8s.Client
// }

// func New(k8sClient *k8s.Client, cfg *config.Configuration, parentCtx context.Context) (*Collector, error) {
// 	ctx, cancel := context.WithCancel(parentCtx)

// 	address := fmt.Sprintf("%s:%s", cfg.Collect.Host, cfg.Collect.Port)
// 	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
// 	if err != nil {
// 		defer cancel()
// 		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
// 	}

// 	client := pb.NewCollectServiceClient(conn)

// 	collector := &Collector{
// 		ctx:       ctx,
// 		doneChan:  make(chan struct{}),
// 		pbc:       client,
// 		k8sClient: k8sClient,
// 	}

// 	go func(c *Collector) {
// 		<-c.ctx.Done() // wait for context to be cancelled
// 		defer cancel()
// 		c.close()
// 	}(collector)

// 	return collector, nil
// }

// func (c *Collector) Start(interval time.Duration, excludedNamespaces []string) error {
// 	ticker := time.NewTicker(interval)
// 	defer ticker.Stop()

// 	// Convert exclude list to map for fast lookup
// 	excludeMap := make(map[string]bool)
// 	for _, ns := range excludedNamespaces {
// 		excludeMap[strings.TrimSpace(ns)] = true
// 	}

// 	ctx := context.Background()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			// This comes from the Deployment manifest
// 			hostname := os.Getenv("NODE_NAME")

// 			summary, err := fetchSummary(hostname)
// 			if err != nil {
// 				log.Printf("summary error: %v", err)
// 				continue
// 			}

// 			var batch []*pb.ContainerMetric

// 			// for each pod collect the utilization metrics
// 			for _, pod := range summary.Pods {
// 				ns := pod.PodRef.Namespace
// 				name := pod.PodRef.Name
// 				for _, container := range pod.Containers {
// 					cpu := float64(container.CPU.UsageNanoCores) / 1e9
// 					mem := container.Memory.UsageBytes
// 					memWS := container.Memory.WorkingSetBytes
// 					fs := container.Rootfs.UsedBytes
// 					rx := container.Network.RxBytes
// 					tx := container.Network.TxBytes

// 					// PodSpec limits
// 					cpuLimit, memLimit := getLimitsForContainer(c.k8sClient.RawClient(), ns, name, container.Name)

// 					batch = append(batch, &pb.ContainerMetric{
// 						Metadata: &pb.MetricMetadata{
// 							ClusterId:     "my-cluster",
// 							NodeName:      hostname,
// 							Namespace:     ns,
// 							PodName:       name,
// 							ContainerName: container.Name,
// 							Source:        "kubelet",
// 							TimestampUtc:  time.Now().UTC().Format(time.RFC3339),
// 						},
// 						CpuUsageCores:         cpu,
// 						MemoryUsageBytes:      mem,
// 						MemoryWorkingSetBytes: memWS,
// 						RxBytes:               rx,
// 						TxBytes:               tx,
// 						FsUsageBytes:          fs,
// 						CpuLimitCores:         cpuLimit,
// 						MemoryLimitBytes:      memLimit,
// 					})
// 				}
// 			}

// 			_, err = c.pbc.SendMetric(ctx, &pb.ContainerMetricBatch{
// 				Metrics: batch,
// 			})
// 			if err != nil {
// 				return err
// 			}
// 			logger.Logger().Info().Msgf("✅ Sent %d metrics", len(batch))
// 		case <-ctx.Done():
// 			logger.Logger().Info().Msg("Shutting down metrics scraper...")
// 			return nil
// 		}
// 	}
// }

// func readToken() (string, error) {
// 	b, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read file: %v", err)
// 	}
// 	return strings.TrimSpace(string(b)), nil
// }

// func fetchSummary(kubeletHost string) (*models.MetricSummary, error) {
// 	url := fmt.Sprintf("https://%s:10250/stats/summary", kubeletHost)
// 	client := &http.Client{
// 		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
// 	}
// 	req, _ := http.NewRequest("GET", url, nil)
// 	token, err := readToken()
// 	if err != nil {
// 		return nil, err
// 	}

// 	req.Header.Set("Authorization", "Bearer "+token)

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var summary models.MetricSummary
// 	err = json.Unmarshal(body, &summary)
// 	return &summary, err
// }

// func getLimitsForContainer(client *kubernetes.Clientset, ns, podName, containerName string) (float64, uint64) {
// 	pod, err := client.CoreV1().Pods(ns).Get(context.Background(), podName, metav1.GetOptions{})
// 	if err != nil {
// 		log.Printf("cannot get pod %s/%s: %v", ns, podName, err)
// 		return 0, 0
// 	}

// 	for _, container := range pod.Spec.Containers {
// 		if container.Name == containerName {
// 			cpu := float64(container.Resources.Limits.Cpu().MilliValue()) / 1000.0
// 			mem := uint64(container.Resources.Limits.Memory().Value())
// 			return cpu, mem
// 		}
// 	}
// 	return 0, 0
// }

// func (c *Collector) Done() <-chan struct{} {
// 	return c.doneChan
// }

// func (c *Collector) close() {
// 	logger.Logger().Info().Msg("MetricCollector closing...")
// }
