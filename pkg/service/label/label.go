package label

import (
	"context"
	"fmt"
	"time"

	"github.com/rss3-network/vsl-reconcile/config"
	"github.com/rss3-network/vsl-reconcile/internal/rpc"
	"github.com/rss3-network/vsl-reconcile/internal/safe"
	"github.com/rss3-network/vsl-reconcile/pkg/kube"
	"github.com/rss3-network/vsl-reconcile/pkg/service"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	labelVSLActive = "vsl.rss3.io/active"
	labelVSLSynced = "vsl.rss3.io/synced"
)

var _ service.Service = (*Service)(nil)

type Service struct {
	name          string
	namespace     string
	checkInterval time.Duration
}

func (s *Service) Run(pool *safe.Pool) error {
	pool.GoCtx(func(ctx context.Context) {
		s.Loop(ctx)
	})

	return nil
}

func (s *Service) Init(cfg *config.Config) error {
	s.name = cfg.DiscoverySTS
	s.namespace = cfg.DiscoveryNS
	s.checkInterval = cfg.CheckInterval

	return nil
}

func (s *Service) String() string {
	return "label"
}

func (s *Service) PodList(ctx context.Context) (*corev1.PodList, error) {
	clientset, err := kube.Client()
	if err != nil {
		return nil, err
	}

	return clientset.CoreV1().Pods(s.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", s.name),
	})
}

func (s *Service) Loop(ctx context.Context) {
	log := zap.L().With(zap.String("service", s.String()))

	clientset, err := kube.Client()
	if err != nil {
		log.Error("failed to initialize kubernetes client", zap.Error(err))
		return
	}

	for {
		pods, err := s.PodList(ctx)
		if err != nil {
			log.Error("failed to list pods", zap.Error(err))
			return
		}

		for _, pod := range pods.Items {
			pod := pod
			url := fmt.Sprintf("http://%s:9545", pod.Status.PodIP)
			isActive, err := rpc.CheckSequencerActive(url)

			if err != nil {
				log.Error("failed to check sequencer active", zap.Error(err))
				continue
			}

			if isActive {
				err = kube.PatchPod(ctx, clientset, s.namespace, pod.Name, labelVSLActive, "true")
				if err != nil {
					log.Error("failed to patch pod", zap.Error(err), zap.String("pod", pod.Name))
				}
			} else {
				err = kube.PatchPod(ctx, clientset, s.namespace, pod.Name, labelVSLActive, "false")
				if err != nil {
					log.Error("failed to patch pod", zap.Error(err), zap.String("pod", pod.Name))
				}
			}
		}

		time.Sleep(s.checkInterval)
	}
}
