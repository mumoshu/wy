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

		restConfig = cluster.RESTConfig()
	}

	return restConfig, nil
}
