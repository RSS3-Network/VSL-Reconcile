package service

import (
	"github.com/rss3-network/vsl-reconcile/config"
	"github.com/rss3-network/vsl-reconcile/internal/safe"
)

// Service defines methods of a service.
type Service interface {
	Run(cfg *config.Config, pool *safe.Pool) error
	Init() error
}
