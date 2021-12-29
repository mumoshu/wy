package argocd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// SecretToCluster converts a secret into a Cluster object
// Derived from https://github.com/argoproj/argo-cd/blob/2147ed3aea727ba128df629d53a1d25fd0f6927c/util/db/cluster.go#L290
func SecretToCluster(s *corev1.Secret) (*Cluster, error) {
	const (
		// AnnotationKeyRefresh is the annotation key which indicates that app needs to be refreshed. Removed by application controller after app is refreshed.
		// Might take values 'normal'/'hard'. Value 'hard' means manifes
		// Copied from https://github.com/argoproj/argo-cd/blob/cc4eea0d6951f1025c9ebb487374658186fa8984/pkg/apis/application/v1alpha1/application_annotations.go#L4-L6
		AnnotationKeyRefresh string = "argocd.argoproj.io/refresh"

		// LabelKeySecretType contains the type of argocd secret (currently: 'cluster', 'repository', 'repo-config' or 'repo-creds')
		// Copied from https://github.com/argoproj/argo-cd/blob/3c874ae065c14102003d041d76d4a337abd72f1e/common/common.go#L107-L108
		LabelKeySecretType = "argocd.argoproj.io/secret-type"

		// AnnotationKeyManagedBy is annotation name which indicates that k8s resource is managed by an application.
		// Copied from https://github.com/argoproj/argo-cd/blob/3c874ae065c14102003d041d76d4a337abd72f1e/common/common.go#L122-L123
		AnnotationKeyManagedBy = "managed-by"
	)

	var config ClusterConfig
	if len(s.Data["config"]) > 0 {
		err := json.Unmarshal(s.Data["config"], &config)
		if err != nil {
			return nil, err
		}
	}

	var namespaces []string
	for _, ns := range strings.Split(string(s.Data["namespaces"]), ",") {
		if ns = strings.TrimSpace(ns); ns != "" {
			namespaces = append(namespaces, ns)
		}
	}
	var refreshRequestedAt *metav1.Time
	if v, found := s.Annotations[AnnotationKeyRefresh]; found {
		requestedAt, err := time.Parse(time.RFC3339, v)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while parsing date in cluster secret '%s': %v\n", s.Name, err)
		} else {
			refreshRequestedAt = &metav1.Time{Time: requestedAt}
		}
	}
	var shard *int64
	if shardStr := s.Data["shard"]; shardStr != nil {
		if val, err := strconv.Atoi(string(shardStr)); err != nil {
			fmt.Fprintf(os.Stderr, "Error while parsing shard in cluster secret '%s': %v\n", s.Name, err)
		} else {
			shard = pointer.Int64Ptr(int64(val))
		}
	}

	// copy labels and annotations excluding system ones
	labels := map[string]string{}
	if s.Labels != nil {
		for k, v := range s.Labels {
			labels[k] = v
		}
		delete(labels, LabelKeySecretType)
	}
	annotations := map[string]string{}
	if s.Annotations != nil {
		for k, v := range s.Annotations {
			annotations[k] = v
		}
		delete(annotations, AnnotationKeyManagedBy)
	}

	cluster := Cluster{
		ID:                 string(s.UID),
		Server:             strings.TrimRight(string(s.Data["server"]), "/"),
		Name:               string(s.Data["name"]),
		Namespaces:         namespaces,
		ClusterResources:   string(s.Data["clusterResources"]) == "true",
		Config:             config,
		RefreshRequestedAt: refreshRequestedAt,
		Shard:              shard,
		Project:            string(s.Data["project"]),
		Labels:             labels,
		Annotations:        annotations,
	}
	return &cluster, nil
}
