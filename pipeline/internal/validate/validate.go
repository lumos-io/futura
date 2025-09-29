package validate

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/opisvigilant/futura/go-lib/stream"
	"github.com/opisvigilant/futura/pipeline/internal/config"
	"github.com/opisvigilant/futura/pipeline/internal/dlq"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
)

const (
	RawEventsTopic  = "raw.k8s.events"
	RawStatsTopic   = "raw.k8s.stats"
	RawObjectsTopic = "raw.k8s.objects"
	RawEBPFTopic    = "raw.ebpf.metrics"

	ValidatedEventsTopic  = "validate.k8s.events"
	ValidatedStatsTopic   = "validate.k8s.stats"
	ValidatedObjectsTopic = "validate.k8s.objects"
	ValidatedEBPFTopic    = "validate.ebpf.metrics"
)

type Validator struct {
	config *config.Configuration
	dlq    *dlq.DLQHandler

	wg sync.WaitGroup
}

func New(config *config.Configuration) (*Validator, error) {
	c, err := dlq.New(config)
	if err != nil {
		return nil, err
	}
	return &Validator{
		config: config,
		dlq:    c,
		wg:     sync.WaitGroup{},
	}, nil
}

func (v *Validator) Start(ctx context.Context) error {
	v.wg.Add(4)

	// Read event messages
	go func() {
		defer v.wg.Done()

		kc, err := stream.NewKafkaClient(v.config.Kafka.Brokers, "validation_group_events")
		if err != nil {
			panic(err)
		}
		log.Info().Msg("Start consuming Kubernete Events...")
		if err := kc.Subscribe(ctx, RawEventsTopic, func(msg stream.Message, ack func() error) {
			m := &pb.KubernetesEvent{}
			if err := proto.Unmarshal(msg.Data(), m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw event message")
				return
			}
			if err := v.ValidateKubernetesEvent(m); err != nil {
				v.dlq.StoreInvalidEventMessage(msg.Data(), err)
				return
			}
			if err := kc.Publish(ctx, ValidatedEventsTopic, msg.Data()); err != nil {
				log.Error().Err(err).Msg("failed to publish the validated event message to the enrichment topic")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", RawEventsTopic)
		}
	}()

	// Read stats messages
	go func() {
		defer v.wg.Done()

		kc, err := stream.NewKafkaClient(v.config.Kafka.Brokers, "validation_group_stats")
		if err != nil {
			panic(err)
		}
		log.Info().Msg("Start consuming Kubernete Kubelet Stats...")
		if err := kc.Subscribe(ctx, RawStatsTopic, func(msg stream.Message, ack func() error) {
			m := &pb.KubernetesKubeletStats{}
			if err := proto.Unmarshal(msg.Data(), m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw stats message")
				return
			}
			if err := v.ValidateKubeletMetrics(m); err != nil {
				v.dlq.StoreInvalidStatMessage(msg.Data(), err)
				return
			}
			if err := kc.Publish(ctx, ValidatedStatsTopic, msg.Data()); err != nil {
				log.Error().Err(err).Msg("failed to publish the validated stats message to the enrichment topic")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", RawStatsTopic)
		}
	}()

	// Read object messages
	go func() {
		defer v.wg.Done()

		kc, err := stream.NewKafkaClient(v.config.Kafka.Brokers, "validation_group_objects")
		if err != nil {
			panic(err)
		}
		log.Info().Msg("Start consuming Kubernete Cluster Object...")
		if err := kc.Subscribe(ctx, RawObjectsTopic, func(msg stream.Message, ack func() error) {
			m := &pb.KubernetesClusterObject{}
			if err := proto.Unmarshal(msg.Data(), m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw object message")
				return
			}
			if err := v.ValidateKubernetesClusterObject(m); err != nil {
				v.dlq.StoreInvalidObjectMessage(msg.Data(), err)
				return
			}
			if err := kc.Publish(ctx, ValidatedObjectsTopic, msg.Data()); err != nil {
				log.Error().Err(err).Msg("failed to publish the validated object message to the enrichment topic")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", RawObjectsTopic)
		}
	}()

	// Read eBPF metrics messages
	go func() {
		defer v.wg.Done()

		kc, err := stream.NewKafkaClient(v.config.Kafka.Brokers, "validation_group_ebpf")
		if err != nil {
			panic(err)
		}
		log.Info().Msg("Start consuming eBPF Metrics...")
		if err := kc.Subscribe(ctx, RawEBPFTopic, func(msg stream.Message, ack func() error) {
			m := &pb.EBPFMetrics{}
			if err := proto.Unmarshal(msg.Data(), m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw eBPF metrics message")
				return
			}
			if err := v.ValidateEBPFMetrics(m); err != nil {
				v.dlq.StoreInvalidEBPFMessage(msg.Data(), err)
				return
			}
			if err := kc.Publish(ctx, ValidatedEBPFTopic, msg.Data()); err != nil {
				log.Error().Err(err).Msg("failed to publish the validated eBPF metrics message to the enrichment topic")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", RawEBPFTopic)
		}
	}()

	v.wg.Wait()

	log.Info().Msg("Ready to say goodbye...")

	return nil
}

// ValidateKubernetesEvents perform sanity checks on KubernetesEvents messages
func (v *Validator) ValidateKubernetesEvent(e *pb.KubernetesEvent) error {
	if e == nil {
		return errors.New("event is nil")
	}
	if e.ObjectKind == "" {
		return errors.New("object_kind is required")
	}
	if e.ObjectName == "" {
		return errors.New("object_name is required")
	}
	if e.ObjectUid == "" {
		return errors.New("object_uid is required")
	}
	if e.ObjectTimestamp <= 0 {
		return fmt.Errorf("invalid object_timestamp: %d", e.ObjectTimestamp)
	}
	if e.EventUid == "" {
		return errors.New("event_uid is required")
	}
	if e.EventSeverityNumber < 0 {
		return fmt.Errorf("event_severity_number must be non-negative: %d", e.EventSeverityNumber)
	}
	if e.EventCount < 0 {
		return fmt.Errorf("event_count must be non-negative: %d", e.EventCount)
	}
	return nil
}

// ValidateKubernetesClusterObject performs checks on KubernetesClusterObject
func (v *Validator) ValidateKubernetesClusterObject(obj *pb.KubernetesClusterObject) error {
	if obj == nil {
		return errors.New("cluster object is nil")
	}
	if obj.Type == "" {
		return errors.New("type is required")
	}
	if obj.Uid == "" {
		return errors.New("uid is required")
	}
	if obj.RestartCount < 0 {
		obj.RestartCount = 0
	}
	// Replica counts must be non-negative
	replicaFields := map[string]int64{
		"replicas":                           obj.Replicas,
		"ready_replicas":                     obj.ReadyReplicas,
		"available_replicas":                 obj.AvailableReplicas,
		"updated_replicas":                   obj.UpdatedReplicas,
		"current_replicas":                   obj.CurrentReplicas,
		"hpa_max_replicas":                   obj.HpaMaxReplicas,
		"hpa_min_replicas":                   obj.HpaMinReplicas,
		"job_active":                         obj.JobActive,
		"job_failed":                         obj.JobFailed,
		"job_succeeded":                      obj.JobSucceeded,
		"job_parallelism":                    obj.JobParallelism,
		"job_completions":                    obj.JobCompletions,
		"ns_phase_value":                     obj.NsPhaseValue,
		"daemonset_current_number_scheduled": obj.DaemonsetCurrentNumberScheduled,
		"daemonset_desired_number_scheduled": obj.DaemonsetDesiredNumberScheduled,
		"daemonset_number_misscheduled":      obj.DaemonsetNumberMisscheduled,
		"daemonset_number_ready":             obj.DaemonsetNumberReady,
	}
	for _, value := range replicaFields {
		if value < 0 {
			value = 0
		}
	}
	return nil
}

// ValidateKubeletMetrics performs sanity checks on KubernetesKubeletStats
func (v *Validator) ValidateKubeletMetrics(m *pb.KubernetesKubeletStats) error {
	if m == nil {
		return errors.New("metrics message is nil")
	}

	if m.Node == nil {
		return errors.New("missing node stats")
	}

	if m.Node.NodeName == "" {
		return errors.New("node name is required")
	}

	for i, c := range m.Node.SystemContainers {
		if err := v.validateContainerStats(c, fmt.Sprintf("systemContainers[%d]", i)); err != nil {
			return err
		}
	}

	for i, pod := range m.Pods {
		if pod.PodRef == nil || pod.PodRef.Uid == "" {
			return fmt.Errorf("pods[%d]: missing podRef.uid", i)
		}
		for j, c := range pod.Containers {
			if err := v.validateContainerStats(c, fmt.Sprintf("pods[%d].containers[%d]", i, j)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *Validator) validateContainerStats(c *pb.ContainerStats, path string) error {
	if c.Name == "" {
		return fmt.Errorf("%s.name is required", path)
	}
	return nil
}

// ValidateEBPFMetrics performs sanity checks on EBPFMetrics
func (v *Validator) ValidateEBPFMetrics(m *pb.EBPFMetrics) error {
	if m == nil {
		return errors.New("eBPF metrics message is nil")
	}

	if m.Timestamp == nil {
		return errors.New("timestamp is required")
	}

	if m.ContainerId == "" {
		return errors.New("container_id is required")
	}

	// Validate that at least one metric type is present
	hasMetrics := m.CpuPatterns != nil ||
		m.MemoryPatterns != nil ||
		m.NetworkFlow != nil ||
		m.Filesystem != nil ||
		m.Http != nil ||
		m.Application != nil ||
		m.Security != nil

	if !hasMetrics {
		return errors.New("at least one metric type is required")
	}

	return nil
}
