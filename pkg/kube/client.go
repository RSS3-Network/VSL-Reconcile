package kube

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func Client() (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	switch {
	case os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "":
		config, err = rest.InClusterConfig()
	case os.Getenv("KUBECONFIG") != "":
		config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	case homedir.HomeDir() != "":
		config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homedir.HomeDir(), ".kube", "config"))
	default:
		return nil, fmt.Errorf("unable to get kubeconfig")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
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
