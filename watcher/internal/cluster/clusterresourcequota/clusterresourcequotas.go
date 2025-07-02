package clusterresourcequota

import (
	"strings"
	"time"

	quotav1 "github.com/openshift/api/quota/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	pbcluster "github.com/opisvigilant/futura/proto/gen/cluster"
)

func RecordMetrics(crq *quotav1.ClusterResourceQuota, ts time.Time) *pbcluster.KubernetesObjectMetadata {
	obj := &pbcluster.KubernetesObjectMetadata{
		Timestamp: timestamppb.New(ts),
		Extra:     make(map[string]string),
	}
	for k, v := range crq.Status.Total.Hard {
		val := extractValue(k, v)
		obj.Extra[string(k)] = string(val)
	}

	for k, v := range crq.Status.Total.Used {
		val := extractValue(k, v)
		obj.Extra[string(k)] = string(val)
	}

	for _, ns := range crq.Status.Namespaces {
		for k, v := range ns.Status.Hard {
			val := extractValue(k, v)
			obj.Extra[string(k)] = string(val)
			mb.RecordOpenshiftAppliedclusterquotaLimitDataPoint(ts, val, ns.Namespace, string(k))
		}

		for k, v := range ns.Status.Used {
			val := extractValue(k, v)
			mb.RecordOpenshiftAppliedclusterquotaUsedDataPoint(ts, val, ns.Namespace, string(k))
		}
	}

	obj.Name = crq.Name
	obj.Uid = string(crq.UID)

	return obj
}

func extractValue(k v1.ResourceName, v resource.Quantity) int64 {
	val := v.Value()
	if strings.HasSuffix(string(k), ".cpu") {
		val = v.MilliValue()
	}
	return val
}
