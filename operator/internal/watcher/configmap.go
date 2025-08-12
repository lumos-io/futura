package watcher

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildWatcherConfigMap(namespace string, apiKey string) *corev1.ConfigMap {
	labels := map[string]string{
		"app.kubernetes.io/name":     "watcher",
		"app.kubernetes.io/instance": "watcher",
	}

	configToml := `[collect]
endpoint = "localhost:50051"
apiKey = "` + apiKey + `"

[kubernetes]
namespaces = []
distribution = "kubernetes"
objectsCollectionInterval = ""
statsCollectionInterval = ""
metadataCollectionInterval = ""
leaseName = "futura"
leaseNamespace = "` + namespace + `"
leaseDuration = ""
renewDeadline = ""
retryPeriod = ""

[kubernetes.auth]
authType = "serviceAccount"
kubeContextName = ""
insecureSkipVerify = true
kubeletCaFile = ""
kubeletCertFile = ""
kubeletKeyFile = ""

[cloud]
provider = "aws"

[log]
level = "debug"
`

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "watcher-config",
			Namespace: namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"config.toml": configToml,
		},
	}
}
