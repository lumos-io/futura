package kubelet

import (
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	"github.com/opisvigilant/futura/watcher/internal/stats/metadata"
	"github.com/opisvigilant/futura/watcher/utils"
)

func addVolumeMetrics(mb *metadata.MetricsBuilder, s stats.VolumeStats, currentTime time.Time) {
	mb.VolumeMetrics.SetCurrentTime(currentTime)
	mb.VolumeMetrics.SetAvailable(utils.PointerToUint64(s.AvailableBytes))
	mb.VolumeMetrics.SetCapacity(utils.PointerToUint64(s.CapacityBytes))
	mb.VolumeMetrics.SetInodes(utils.PointerToUint64(s.Inodes))
	mb.VolumeMetrics.SetInodesFree(utils.PointerToUint64(s.InodesFree))
	mb.VolumeMetrics.SetInodesUsed(utils.PointerToUint64(s.InodesUsed))
}

func setResourcesFromVolume(rb *metadata.ResourceBuilder, volume v1.Volume) {
	switch {
	// TODO: Support more types
	case volume.ConfigMap != nil:
		rb.SetK8sVolumeType(labelValueConfigMapVolume)
	case volume.DownwardAPI != nil:
		rb.SetK8sVolumeType(labelValueDownwardAPIVolume)
	case volume.EmptyDir != nil:
		rb.SetK8sVolumeType(labelValueEmptyDirVolume)
	case volume.Secret != nil:
		rb.SetK8sVolumeType(labelValueSecretVolume)
	case volume.PersistentVolumeClaim != nil:
		rb.SetK8sVolumeType(labelValuePersistentVolumeClaim)
		rb.SetK8sPersistentvolumeclaimName(volume.PersistentVolumeClaim.ClaimName)
	case volume.HostPath != nil:
		rb.SetK8sVolumeType(labelValueHostPathVolume)
	case volume.AWSElasticBlockStore != nil:
		awsElasticBlockStoreDims(rb, *volume.AWSElasticBlockStore)
	case volume.GCEPersistentDisk != nil:
		gcePersistentDiskDims(rb, *volume.GCEPersistentDisk)
	}
}

func SetPersistentVolumeLabels(rb *metadata.ResourceBuilder, pv v1.PersistentVolumeSource) {
	switch {
	case pv.Local != nil:
		rb.SetK8sVolumeType(labelValueLocalVolume)
	case pv.AWSElasticBlockStore != nil:
		awsElasticBlockStoreDims(rb, *pv.AWSElasticBlockStore)
	case pv.GCEPersistentDisk != nil:
		gcePersistentDiskDims(rb, *pv.GCEPersistentDisk)
	}
}

func awsElasticBlockStoreDims(rb *metadata.ResourceBuilder, vs v1.AWSElasticBlockStoreVolumeSource) {
	rb.SetK8sVolumeType(labelValueAWSEBSVolume)
	// AWS specific labels.
	rb.SetAwsVolumeID(vs.VolumeID)
	rb.SetFsType(vs.FSType)
	rb.SetPartition(strconv.Itoa(int(vs.Partition)))
}

func gcePersistentDiskDims(rb *metadata.ResourceBuilder, vs v1.GCEPersistentDiskVolumeSource) {
	rb.SetK8sVolumeType(labelValueGCEPDVolume)
	// GCP specific labels.
	rb.SetGcePdName(vs.PDName)
	rb.SetFsType(vs.FSType)
	rb.SetPartition(strconv.Itoa(int(vs.Partition)))
}
