package resourcequota

import (
	"strings"
	"time"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
)

func RecordMetrics(rq *corev1.ResourceQuota, ts time.Time) *pbcluster.KubernetesObjectMetadata {
	obj := &pbcluster.KubernetesObjectMetadata{
		Timestamp: timestamppb.New(ts),
	}
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

	return obj
}
