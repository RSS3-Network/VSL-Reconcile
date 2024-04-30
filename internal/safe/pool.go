package safe

import (
	"context"
	"sync"
)

type routineCtx func(ctx context.Context)

// Pool is a pool of go routines.
type Pool struct {
	waitGroup sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewPool creates a Pool.
func NewPool(parentCtx context.Context) *Pool {
	ctx, cancel := context.WithCancel(parentCtx)

	return &Pool{
		ctx:    ctx,
		cancel: cancel,
	}
}

// GoCtx starts a recoverable goroutine with a context.
func (p *Pool) GoCtx(goroutine routineCtx) {
	p.waitGroup.Add(1)
	Go(func() {
		defer p.waitGroup.Done()
		goroutine(p.ctx)
	})
}

// Stop stops all started routines, waiting for their termination.
func (p *Pool) Stop() {
	p.cancel()
	p.waitGroup.Wait()
}
