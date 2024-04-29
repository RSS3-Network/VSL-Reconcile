package config

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	EndpointProtocol = "http"
	EndpointsPort    = 9545

	ClusterLocalSuffix = ".svc.cluster.local"
)

func initKubeClient() (*kubernetes.Clientset, error) {
	var kubeconfig *string

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()

	var config *rest.Config
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	// create the clientset
	return kubernetes.NewForConfig(config)
}

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

		ep := fmt.Sprintf("%s://%s.%s.%s.svc.cluster.local:%d",
			EndpointProtocol, podName, svcName, namespace, EndpointsPort,
		)

		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}
