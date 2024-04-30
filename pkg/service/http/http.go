package http

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/rss3-network/vsl-reconcile/config"
	"github.com/rss3-network/vsl-reconcile/internal/safe"
	"github.com/rss3-network/vsl-reconcile/pkg/service"
)

var _ service.Service = (*Service)(nil)

type Service struct {
	server *echo.Echo
}

func (s *Service) Run(pool *safe.Pool) error {
	pool.GoCtx(func(ctx context.Context) {
		err := s.server.Start(":8080")
		if err != nil {
			ctx.Done()
		}
	})

	return nil
}

func (s *Service) Init(_ *config.Config) error {
	s.server = echo.New()
	s.server.HideBanner = true
	s.server.HidePort = true
	s.server.GET("/", func(c echo.Context) error {
		return c.String(200, "Hello, World!")
	})

	return nil
}

func (s *Service) String() string {
	return "http"
}
