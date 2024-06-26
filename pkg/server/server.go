package server

import (
	"github.com/rss3-network/vsl-reconcile/internal/safe"
	"github.com/rss3-network/vsl-reconcile/pkg/service"
	"go.uber.org/zap"
)

type Server struct {
	serviceAggregator service.Service
	routinesPool      *safe.Pool
	stopChan          chan bool
}

func NewServer(svc service.Service, pool *safe.Pool) *Server {
	s := &Server{
		serviceAggregator: svc,
		routinesPool:      pool,
		stopChan:          make(chan bool, 1),
	}

	return s
}

func (s *Server) Start() {
	s.startAggregator()
}

func (s *Server) Wait() {
	<-s.stopChan
}

func (s *Server) Stop() {
	s.stopChan <- true
}

func (s *Server) startAggregator() {
	safe.Go(func() {
		err := s.serviceAggregator.Run(s.routinesPool)
		if err != nil {
			zap.L().Error("service aggregator failed",
				zap.Error(err),
				zap.String("service", s.serviceAggregator.String()),
			)
		}
	})
}
