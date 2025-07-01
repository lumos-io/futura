package cluster

import (
	"github.com/opisvigilant/futura/watcher/internal/config"
)

// type metadataConsumer func(metadata []*experimentalmetricmetadata.MetadataUpdate) error

type KubernetesClusterCollector struct {
}

func New(config *config.Configuration) (*KubernetesClusterCollector, error) {
	return &KubernetesClusterCollector{}, nil
}

/*
k8sClient, err := kubernetes.MakeClient(kubernetes.APIConfig{
		AuthType: kubernetes.AuthType(kec.config.Kubernetes.AuthType),
		Context:  kec.config.Kubernetes.KubeContextName,
	})
*/
