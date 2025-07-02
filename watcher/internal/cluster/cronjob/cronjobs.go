package cronjob

import (
	"time"

	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	batchv1 "k8s.io/api/batch/v1"
)

const (
	// Keys for cronjob metadata.
	cronJobKeySchedule          = "schedule"
	cronJobKeyConcurrencyPolicy = "concurrency_policy"
)

func RecordMetrics(mb *metadata.MetricsBuilder, cj *batchv1.CronJob, ts time.Time) {
	mb.RecordK8sCronjobActiveJobsDataPoint(ts, int64(len(cj.Status.Active)))

	rb := mb.NewResourceBuilder()
	rb.SetK8sNamespaceName(cj.Namespace)
	rb.SetK8sCronjobUID(string(cj.UID))
	rb.SetK8sCronjobName(cj.Name)
	mb.EmitForResource(metadata.WithResource(rb.Emit()))
}

func GetMetadata(cj *batchv1.CronJob) map[metadata.ResourceID]*metadata.KubernetesMetadata {
	rm := metadata.GetGenericMetadata(&cj.ObjectMeta, constants.K8sKindCronJob)
	rm.Metadata[cronJobKeySchedule] = cj.Spec.Schedule
	rm.Metadata[cronJobKeyConcurrencyPolicy] = string(cj.Spec.ConcurrencyPolicy)
	return map[metadata.ResourceID]*metadata.KubernetesMetadata{metadata.ResourceID(cj.UID): rm}
}
