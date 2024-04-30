package service

import (
	"github.com/rss3-network/vsl-reconcile/config"
	"github.com/rss3-network/vsl-reconcile/internal/safe"
)

// Service defines methods of a service.
type Service interface {
	Run(pool *safe.Pool) error
	Init(cfg *config.Config) error
	String() string
}
