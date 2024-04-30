package server

import (
	"log"

	"github.com/rss3-network/vsl-reconcile/config"
	"github.com/rss3-network/vsl-reconcile/internal/safe"
	"github.com/rss3-network/vsl-reconcile/pkg/service"
)

type Server struct {
	serviceAggregator service.Service
	routinesPool      *safe.Pool
	config            *config.Config
	stopChan          chan bool
}

func NewServer(svc service.Service, cfg *config.Config, pool *safe.Pool) *Server {
	s := &Server{
		serviceAggregator: svc,
		routinesPool:      pool,
		config:            cfg,
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
		err := s.serviceAggregator.Run(s.config, s.routinesPool)
		if err != nil {
			log.Fatalf("Error starting service aggregator: %v", err)
		}
	})
}
