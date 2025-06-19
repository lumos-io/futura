package k8s

import (
	"flag"
	"fmt"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Client struct {
	clientSet *kubernetes.Clientset
}

func New(inCluster bool) (*Client, error) {
	// get incluster kubeconfig
	var kubeconfig *string
	var kubeConfig *rest.Config

	if !inCluster {
		var err error
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		} else {
			kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
		}

		flag.Parse()

		kubeConfig, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, err
		}
	} else {
		// in cluster config, default
		var err error
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to get incluster kubeconfig: %w", err)
		}
	}

	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create kubeClient: %w", err)
	}

	return &Client{
		clientSet: kubeClient,
	}, nil
}

func (c *Client) RawClient() *kubernetes.Clientset {
	return c.clientSet
}

func (c *Client) GetVersion() (string, error) {
	version, err := c.clientSet.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("unable to get k8s server version: %w", err)
	}
	return version.String(), nil
}
