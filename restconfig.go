package main

import (
	"context"
	"strings"

	"github.com/mumoshu/wy/pkg/argocd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/exec"
	"k8s.io/client-go/rest"
)

func getRestConfig(kubeconfig string, argocdClusterSecret string) (*rest.Config, error) {
	restConfig, err := argocd.NewRestConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	if argocdClusterSecret != "" {
		cluster, err := getCluster(restConfig, argocdClusterSecret)
		if err != nil {
			return nil, err
		}

		// cluster.RestConfig() rewrites the exec provider and tls config in order to
		// force the use of a custom transport.
		restConfig = cluster.RESTConfig()
	}

	return restConfig, nil
}

func getKubeconfig(kubeconfig string, argocdClusterSecret string, setNamespace string) ([]byte, error) {
	restConfig, err := argocd.NewRestConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	cluster, err := getCluster(restConfig, argocdClusterSecret)
	if err != nil {
		return nil, err
	}

	//
	// We intentionally don't use cluster.RestConfig() to avoid emptying
	// important details written in the kubeconfig, like exec credentials, tls client config, and auth provider.
	//
	// cluster.RestConfig() is basically cluster.RawRestConfig() followed by SetK8SConfigDefaults()
	// so the difference is the avoidance of SetK8SConfigDefaults().
	// SetK8SConfigDefaults() removes the config details as explained in the first paragraph hence
	// this keeps the details.

	clusterRestConfig := cluster.RawRestConfig()

	kubeconfigData, err := argocd.GenerateKubeConfiguration(clusterRestConfig, setNamespace)
	if err != nil {
		return nil, err
	}

	return kubeconfigData, nil
}

func getCluster(restConfig *rest.Config, argocdClusterSecret string) (*argocd.Cluster, error) {
	c, err := argocd.NewClientSet(restConfig)
	if err != nil {
		return nil, err
	}

	ctx := context.TODO()

	nsName := strings.Split(argocdClusterSecret, "/")

	var ns, name string

	if len(nsName) == 1 {
		ns = "default"
		name = nsName[0]
	} else {
		ns = nsName[0]
		name = nsName[1]
	}

	secret, err := c.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	cluster, err := argocd.SecretToCluster(secret)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}
