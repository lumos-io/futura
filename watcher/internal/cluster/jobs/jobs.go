package jobs

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	batchv1 "k8s.io/api/batch/v1"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"
	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
)

func RecordMetrics(j *batchv1.Job, ts time.Time) *pbcluster.KubernetesObjectMetadata {
	obj := &pbcluster.KubernetesObjectMetadata{
		Timestamp: timestamppb.New(ts),
		Namespace: j.Namespace,
		Name:      j.Name,
		Uid:       string(j.UID),
	}

	mb.RecordK8sJobActivePodsDataPoint(ts, int64(j.Status.Active))
	mb.RecordK8sJobFailedPodsDataPoint(ts, int64(j.Status.Failed))
	mb.RecordK8sJobSuccessfulPodsDataPoint(ts, int64(j.Status.Succeeded))

	if j.Spec.Completions != nil {
		mb.RecordK8sJobDesiredSuccessfulPodsDataPoint(ts, int64(*j.Spec.Completions))
	}
	if j.Spec.Parallelism != nil {
		mb.RecordK8sJobMaxParallelPodsDataPoint(ts, int64(*j.Spec.Parallelism))
	}

	return obj
}

// Transform transforms the job to remove the fields that we don't use to reduce RAM utilization.
// IMPORTANT: Make sure to update this function before using new job fields.
func Transform(job *batchv1.Job) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metadata.TransformObjectMeta(job.ObjectMeta),
		Spec: batchv1.JobSpec{
			Completions: job.Spec.Completions,
			Parallelism: job.Spec.Parallelism,
		},
		Status: batchv1.JobStatus{
			Active:    job.Status.Active,
			Succeeded: job.Status.Succeeded,
			Failed:    job.Status.Failed,
		},
	}
}

func GetMetadata(j *batchv1.Job) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{
		metadata.ResourceID(j.UID): metadata.GetGenericMetadata(&j.ObjectMeta, constants.K8sKindJob),
	}
}
