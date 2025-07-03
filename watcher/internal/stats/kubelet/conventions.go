package kubelet

const (
	labelVolumeType = "k8s.volume.type"

	// Volume types.
	labelValuePersistentVolumeClaim = "persistentVolumeClaim"
	labelValueConfigMapVolume       = "configMap"
	labelValueDownwardAPIVolume     = "downwardAPI"
	labelValueEmptyDirVolume        = "emptyDir"
	labelValueSecretVolume          = "secret"
	labelValueHostPathVolume        = "hostPath"
	labelValueLocalVolume           = "local"
	labelValueAWSEBSVolume          = "awsElasticBlockStore"
	labelValueGCEPDVolume           = "gcePersistentDisk"
	labelValueGlusterFSVolume       = "glusterfs"
)
