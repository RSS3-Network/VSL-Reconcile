package heartbeat

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	EndpointProtocol = "http"
	EndpointsPort    = 9545

	ClusterLocalSuffix = "cluster.local"
)

// DiscoverStsEndpoints : Discover StatefulSet endpoints
func DiscoverStsEndpoints(clientset *kubernetes.Clientset, name, namespace string) ([]string, error) {
	ctx := context.Background()

	sts, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// get headless service
	svcName := sts.Spec.ServiceName

	// get headless service pod dns
	var endpoints []string

	for i := 0; i < int(*sts.Spec.Replicas); i++ {
		podName := fmt.Sprintf("%s-%d", name, i)

		ep := fmt.Sprintf("%s://%s.%s.%s.svc.%s:%d",
			EndpointProtocol, podName, svcName, namespace, ClusterLocalSuffix, EndpointsPort,
		)

		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}
