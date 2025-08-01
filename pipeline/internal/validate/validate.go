package validate

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/opisvigilant/futura/go-lib/stream"
	"github.com/opisvigilant/futura/pipeline/internal/config"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbcl "github.com/opisvigilant/futura/proto/gen/cluster"
	pbev "github.com/opisvigilant/futura/proto/gen/events"
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
)

const (
	RawEventsTopic  = "raw.k8s.events"
	RawStatsTopic   = "raw.k8s.stats"
	RawObjectsTopic = "raw.k8s.objects"

	ValidatedEventsTopic  = "validate.k8s.events"
	ValidatedStatsTopic   = "validate.k8s.stats"
	ValidatedObjectsTopic = "validate.k8s.objects"
)

type Validator struct {
	stream stream.Stream

	wg sync.WaitGroup
}

func New(config *config.Configuration) (*Validator, error) {
	kc, err := stream.NewKafkaClient(config.Kafka.Brokers, "validation_group")
	if err != nil {
		return nil, err
	}
	return &Validator{
		stream: kc,
		wg:     sync.WaitGroup{},
	}, nil
}

func (v *Validator) Start(ctx context.Context) error {
	v.wg.Add(3)

	// Read event messages
	go func() {
		defer v.wg.Done()

		if err := v.stream.Subscribe(ctx, RawEventsTopic, func(msg stream.Message, ack func() error) {
			var m *pbev.KubernetesEvent
			if err := protojson.Unmarshal(msg.Data(), m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw event message")
				return
			}
			if err := v.ValidateKubernetesEvent(m); err != nil {
				log.Error().Err(err).Msg("failed to validate the raw event message")
				return
			}
			if err := v.stream.Publish(ctx, ValidatedEventsTopic, msg.Data()); err != nil {
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

		if err := v.stream.Subscribe(ctx, RawStatsTopic, func(msg stream.Message, ack func() error) {
			var m *pbst.KubernetesKubeletMetrics
			if err := protojson.Unmarshal(msg.Data(), m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw stats message")
				return
			}
			if err := v.ValidateKubeletMetrics(m); err != nil {
				log.Error().Err(err).Msg("failed to validate the raw stats message")
				return
			}
			if err := v.stream.Publish(ctx, ValidatedStatsTopic, msg.Data()); err != nil {
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

		if err := v.stream.Subscribe(ctx, RawObjectsTopic, func(msg stream.Message, ack func() error) {
			var m *pbcl.KubernetesClusterObject
			if err := protojson.Unmarshal(msg.Data(), m); err != nil {
				log.Error().Err(err).Msg("failed to proto-unmarshal the raw object message")
				return
			}
			if err := v.ValidateKubernetesClusterObject(m); err != nil {
				log.Error().Err(err).Msg("failed to validate the raw object message")
				return
			}
			if err := v.stream.Publish(ctx, ValidatedObjectsTopic, msg.Data()); err != nil {
				log.Error().Err(err).Msg("failed to publish the validated object message to the enrichment topic")
				return
			}
		}); err != nil {
			log.Error().Err(err).Msgf("failed to subscribe to stream `%s`", RawObjectsTopic)
		}
	}()

	v.wg.Wait()

	return nil
}

// ValidateKubernetesEvents perform sanity checks on KubernetesEvents messages
func (v *Validator) ValidateKubernetesEvent(e *pbev.KubernetesEvent) error {
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

	// Validate object_timestamp is within ±24h
	objTime := time.Unix(e.ObjectTimestamp, 0)
	now := time.Now()
	if objTime.Before(now.Add(-24*time.Hour)) || objTime.After(now.Add(24*time.Hour)) {
		return fmt.Errorf("object_timestamp out of acceptable range: %s", objTime)
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

	// Optionally parse event_starttime if format is known (e.g., RFC3339)
	if e.EventStarttime != "" {
		if _, err := time.Parse(time.RFC3339, e.EventStarttime); err != nil {
			return fmt.Errorf("event_starttime not RFC3339: %s", e.EventStarttime)
		}
	}

	return nil
}

// ValidateKubernetesClusterObject performs checks on KubernetesClusterObject
func (v *Validator) ValidateKubernetesClusterObject(obj *pbcl.KubernetesClusterObject) error {
	if obj == nil {
		return errors.New("cluster object is nil")
	}

	if err := v.validateTimestampPB(obj.Timestamp, "timestamp"); err != nil {
		return err
	}
	if obj.Type == "" {
		return errors.New("type is required")
	}
	if obj.Kind == "" {
		return errors.New("kind is required")
	}
	if obj.Name == "" {
		return errors.New("name is required")
	}
	if obj.Uid == "" {
		return errors.New("uid is required")
	}

	if obj.RestartCount < 0 {
		return fmt.Errorf("restart_count must be non-negative")
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
	for field, value := range replicaFields {
		if value < 0 {
			return fmt.Errorf("%s must be non-negative", field)
		}
	}

	for i, c := range obj.Containers {
		if c.Name == "" {
			return fmt.Errorf("containers[%d].name is required", i)
		}
		if c.Image == "" {
			return fmt.Errorf("containers[%d].image is required", i)
		}
		if c.RestartsCount < 0 {
			return fmt.Errorf("containers[%d].restarts_count is negative", i)
		}
	}

	for i, v := range obj.Volumes {
		if v.Name == "" {
			return fmt.Errorf("volumes[%d].name is required", i)
		}
		if v.Type == "" {
			return fmt.Errorf("volumes[%d].type is required", i)
		}
	}

	for i, cond := range obj.Conditions {
		if cond.Type == "" {
			return fmt.Errorf("conditions[%d].type is required", i)
		}
		if cond.Status == "" {
			return fmt.Errorf("conditions[%d].status is required", i)
		}
	}

	if obj.ClusterQuota != nil {
		if obj.ClusterQuota.Name == "" {
			return errors.New("cluster_quota.name is required")
		}
		if obj.ClusterQuota.Uid == "" {
			return errors.New("cluster_quota.uid is required")
		}
	}

	return nil
}

func (v *Validator) validateTimestampPB(ts *timestamppb.Timestamp, field string) error {
	if ts == nil {
		return fmt.Errorf("%s is nil", field)
	}
	t := ts.AsTime()
	now := time.Now()
	if t.Before(now.Add(-24*time.Hour)) || t.After(now.Add(24*time.Hour)) {
		return fmt.Errorf("%s timestamp out of range: %s", field, t)
	}
	return nil
}

// ValidateKubeletMetrics performs sanity checks on KubernetesKubeletMetrics
func (v *Validator) ValidateKubeletMetrics(m *pbst.KubernetesKubeletMetrics) error {
	if m == nil {
		return errors.New("metrics message is nil")
	}

	if m.Node == nil {
		return errors.New("missing node stats")
	}

	if m.Node.NodeName == "" {
		return errors.New("node name is required")
	}

	if err := v.validateTimestampPB(m.Node.StartTime, "node.startTime"); err != nil {
		return err
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
		if err := v.validateTimestampPB(pod.StartTime, fmt.Sprintf("pods[%d].startTime", i)); err != nil {
			return err
		}
		for j, c := range pod.Containers {
			if err := v.validateContainerStats(c, fmt.Sprintf("pods[%d].containers[%d]", i, j)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *Validator) validateContainerStats(c *pbst.ContainerStats, path string) error {
	if c.Name == "" {
		return fmt.Errorf("%s.name is required", path)
	}
	if err := v.validateTimestampPB(c.StartTime, path+".startTime"); err != nil {
		return err
	}
	return nil
}
