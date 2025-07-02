package resourcequota

import (
	"strings"
	"time"

	"github.com/opisvigilant/futura/watcher/internal/cluster/metadata"
	corev1 "k8s.io/api/core/v1"
)

func RecordMetrics(mb *metadata.MetricsBuilder, rq *corev1.ResourceQuota, ts time.Time) {
	for k, v := range rq.Status.Hard {
		val := v.Value()
		if strings.HasSuffix(string(k), ".cpu") {
			val = v.MilliValue()
		}
		mb.RecordK8sResourceQuotaHardLimitDataPoint(ts, val, string(k))
	}

	for k, v := range rq.Status.Used {
		val := v.Value()
		if strings.HasSuffix(string(k), ".cpu") {
			val = v.MilliValue()
		}
		mb.RecordK8sResourceQuotaUsedDataPoint(ts, val, string(k))
	}

	rb := mb.NewResourceBuilder()
	rb.SetK8sResourcequotaUID(string(rq.UID))
	rb.SetK8sResourcequotaName(rq.Name)
	rb.SetK8sNamespaceName(rq.Namespace)
	mb.EmitForResource(metadata.WithResource(rb.Emit()))
}
