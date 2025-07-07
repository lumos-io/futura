package metadata

import (
	pbst "github.com/opisvigilant/futura/proto/gen/stats"
	"github.com/opisvigilant/futura/watcher/internal/config"
)

type ResourceBuilder struct {
	config   *config.Configuration
	resource *pbst.Resource
}

func NewResourceBuilder(config *config.Configuration) *ResourceBuilder {
	return &ResourceBuilder{
		config: config,
		resource: &pbst.Resource{
			Attributes: make(map[string]string),
		},
	}
}

func (rb *ResourceBuilder) Emit() *pbst.Resource {
	return rb.resource
}

// SetAwsVolumeID sets provided value as "aws.volume.id" attribute.
func (rb *ResourceBuilder) SetAwsVolumeID(val string) {
	if rb.config.Cloud.Provider == "aws" {
		rb.resource.Attributes["aws.volume.id"] = val
	}
}

// SetContainerID sets provided value as "container.id" attribute.
func (rb *ResourceBuilder) SetContainerID(val string) {
	rb.resource.Attributes["container.id"] = val
}

// SetFsType sets provided value as "fs.type" attribute.
func (rb *ResourceBuilder) SetFsType(val string) {
	rb.resource.Attributes["fs.type"] = val
}

// SetGcePdName sets provided value as "gce.pd.name" attribute.
func (rb *ResourceBuilder) SetGcePdName(val string) {
	if rb.config.Cloud.Provider == "gcp" {
		rb.resource.Attributes["gce.pd.name"] = val
	}
}

// SetK8sContainerName sets provided value as "k8s.container.name" attribute.
func (rb *ResourceBuilder) SetK8sContainerName(val string) {
	rb.resource.Attributes["k8s.container.name"] = val
}

// SetK8sNamespaceName sets provided value as "k8s.namespace.name" attribute.
func (rb *ResourceBuilder) SetK8sNamespaceName(val string) {
	rb.resource.Attributes["k8s.namespace.name"] = val
}

// SetK8sNodeName sets provided value as "k8s.node.name" attribute.
func (rb *ResourceBuilder) SetK8sNodeName(val string) {
	rb.resource.Attributes["k8s.node.name"] = val
}

// SetK8sPersistentvolumeclaimName sets provided value as "k8s.persistentvolumeclaim.name" attribute.
func (rb *ResourceBuilder) SetK8sPersistentvolumeclaimName(val string) {
	rb.resource.Attributes["k8s.persistentvolumeclaim.name"] = val
}

// SetK8sPodName sets provided value as "k8s.pod.name" attribute.
func (rb *ResourceBuilder) SetK8sPodName(val string) {
	rb.resource.Attributes["k8s.pod.name"] = val
}

// SetK8sPodUID sets provided value as "k8s.pod.uid" attribute.
func (rb *ResourceBuilder) SetK8sPodUID(val string) {
	rb.resource.Attributes["k8s.pod.uid"] = val
}

// SetK8sVolumeName sets provided value as "k8s.volume.name" attribute.
func (rb *ResourceBuilder) SetK8sVolumeName(val string) {
	rb.resource.Attributes["k8s.volume.name"] = val
}

// SetK8sVolumeType sets provided value as "k8s.volume.type" attribute.
func (rb *ResourceBuilder) SetK8sVolumeType(val string) {
	rb.resource.Attributes["k8s.volume.type"] = val
}

// SetPartition sets provided value as "partition" attribute.
func (rb *ResourceBuilder) SetPartition(val string) {
	rb.resource.Attributes["partition"] = val
}
