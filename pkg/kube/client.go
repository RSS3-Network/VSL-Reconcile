package kube

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func Client() (*kubernetes.Clientset, error) {
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

// PatchPod patches a pod with a label.
func PatchPod(ctx context.Context, clientset *kubernetes.Clientset, namespace, name, key, value string) error {
	patch := fmt.Sprintf(`{"metadata":{"labels":{"%s":"%s"}}}`, key, value)

	_, err := clientset.CoreV1().Pods(namespace).Patch(ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})

	return err
}
