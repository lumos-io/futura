package cronjob

import (
	"time"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"
	constants "github.com/opisvigilant/futura/watcher/internal/cluster/constants"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	batchv1 "k8s.io/api/batch/v1"
)

const (
	// Keys for cronjob metadata.
	cronJobKeySchedule          = "schedule"
	cronJobKeyConcurrencyPolicy = "concurrency_policy"
)

func RecordMetrics(cj *batchv1.CronJob, ts time.Time) *pbcluster.KubernetesClusterObject {
	obj := &pbcluster.KubernetesClusterObject{
		Timestamp: timestamppb.New(ts),
		Kind:      cj.Kind,
		Namespace: cj.Namespace,
		Uid:       string(cj.UID),
		Name:      cj.Name,
	}

	// TODO: how do I store the active cronjobs??
	// mb.RecordK8sCronjobActiveJobsDataPoint(ts, int64(len(cj.Status.Active)))

	return obj
}

func GetMetadata(cj *batchv1.CronJob) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	rm := metadata.GetGenericMetadata(&cj.ObjectMeta, constants.K8sKindCronJob)
	rm.Metadata[cronJobKeySchedule] = cj.Spec.Schedule
	rm.Metadata[cronJobKeyConcurrencyPolicy] = string(cj.Spec.ConcurrencyPolicy)
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{metadata.ResourceID(cj.UID): rm}
}
