package resourcequota

import (
	"fmt"
	"strings"
	"time"

	pb "github.com/opisvigilant/futura/proto/gen/telemetry"
	"google.golang.org/protobuf/types/known/timestamppb"
	corev1 "k8s.io/api/core/v1"
)

func RecordMetrics(rq *corev1.ResourceQuota, ts time.Time) *pb.KubernetesClusterObject {
	obj := &pb.KubernetesClusterObject{
		Timestamp: timestamppb.New(ts),
		Namespace: rq.Namespace,
		Name:      rq.Name,
		Uid:       string(rq.UID),
		Extra:     make(map[string]string),
	}
	for k, v := range rq.Status.Hard {
		val := v.Value()
		if strings.HasSuffix(string(k), ".cpu") {
			val = v.MilliValue()
		}
		obj.Extra[string(k)] = fmt.Sprint(val)
	}

	for k, v := range rq.Status.Used {
		val := v.Value()
		if strings.HasSuffix(string(k), ".cpu") {
			val = v.MilliValue()
		}
		obj.Extra[string(k)] = fmt.Sprint(val)
	}

	return obj
}
