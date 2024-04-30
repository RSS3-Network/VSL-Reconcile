package aggregator

import (
	"context"
	"log"

	"github.com/rss3-network/vsl-reconcile/config"
	"github.com/rss3-network/vsl-reconcile/internal/safe"
	"github.com/rss3-network/vsl-reconcile/pkg/service"
)

var _ service.Service = (*ServiceAggregator)(nil)

// ServiceAggregator aggregates services.
type ServiceAggregator struct {
	services []service.Service
}

func New(cfg *config.Config, services ...service.Service) *ServiceAggregator {
	s := &ServiceAggregator{}

	for _, svc := range services {
		err := s.AddService(cfg, svc)
		if err != nil {
			log.Printf("Error adding service: %v", err)
		}
	}

	return s
}

// AddService adds a service to the aggregator.
func (s *ServiceAggregator) AddService(cfg *config.Config, svc service.Service) error {
	err := svc.Init(cfg)
	if err != nil {
		return err
	}

	s.services = append(s.services, svc)

	return nil
}

func (s *ServiceAggregator) Run(pool *safe.Pool) error {
	for _, svc := range s.services {
		svc := svc

		safe.Go(func() {
			s.startService(pool, svc)
		})
	}

	return nil
}

func (s *ServiceAggregator) Init(_ *config.Config) error {
	return nil
}

func (s *ServiceAggregator) startService(pool *safe.Pool, svc service.Service) {
	pool.GoCtx(func(ctx context.Context) {
		for range ctx.Done() {
			return
		}
	})

	if err := svc.Run(pool); err != nil {
		log.Printf("Error running service: %v", err)
	}
}
