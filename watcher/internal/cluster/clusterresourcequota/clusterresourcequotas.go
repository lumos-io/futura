package clusterresourcequota

import (
	"strings"
	"time"

	quotav1 "github.com/openshift/api/quota/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
)

func RecordMetrics(crq *quotav1.ClusterResourceQuota, ts time.Time) *pb.KubernetesClusterObject {
	obj := &pb.KubernetesClusterObject{
		Timestamp: timestamppb.New(ts),
		Name:      crq.Name,
		Uid:       string(crq.UID),
	}

	clusterQuota := &pb.ClusterResourceQuotaMetadata{
		Name: crq.Name,
		Uid:  string(crq.UID),
	}

	for k, v := range crq.Status.Total.Hard {
		val := extractValue(k, v)
		clusterQuota.TotalLimits = append(clusterQuota.TotalLimits, &pb.QuotaResource{
			Resource: string(k),
			Value:    val,
		})
	}

	for k, v := range crq.Status.Total.Used {
		val := extractValue(k, v)
		clusterQuota.TotalUsage = append(clusterQuota.TotalUsage, &pb.QuotaResource{
			Resource: string(k),
			Value:    val,
		})
	}

	for _, ns := range crq.Status.Namespaces {
		nsQuota := &pb.NamespaceQuota{
			Namespace: ns.Namespace,
		}

		for k, v := range ns.Status.Hard {
			val := extractValue(k, v)
			nsQuota.Limits = append(nsQuota.Limits, &pb.QuotaResource{
				Resource: string(k),
				Value:    val,
			})
		}

		for k, v := range ns.Status.Used {
			val := extractValue(k, v)
			nsQuota.Usage = append(nsQuota.Usage, &pb.QuotaResource{
				Resource: string(k),
				Value:    val,
			})
		}

		clusterQuota.Quotas = append(clusterQuota.Quotas, nsQuota)
	}

	obj.ClusterQuota = clusterQuota

	return obj
}

func extractValue(k v1.ResourceName, v resource.Quantity) int64 {
	val := v.Value()
	if strings.HasSuffix(string(k), ".cpu") {
		val = v.MilliValue()
	}
	return val
}
