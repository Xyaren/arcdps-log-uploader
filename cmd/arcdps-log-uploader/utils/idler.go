package utils

import (
	"context"
	"sync"
	"time"
)

type Idler struct {
	mu             sync.Mutex
	cancel         context.CancelFunc
	invoke         func()
	idleTimeout    time.Duration
	lastInvocation time.Time
}

func NewIdler(idleTimeout time.Duration, invoke func()) Idler {
	return Idler{invoke: invoke, idleTimeout: idleTimeout}
}

func (i *Idler) Call() {
	i.mu.Lock()
	defer i.mu.Unlock()

	now := time.Now()
	if !i.lastInvocation.Add(i.idleTimeout).After(now) {
		i.lastInvocation = now

		go i.invoke()
		return
	}

	if i.cancel != nil {
		i.cancel()
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	i.cancel = cancelFunc
	go i.runIn(ctx, i.idleTimeout)
}

func (i *Idler) runIn(ctx context.Context, idleTime time.Duration) {
	select {
	case <-time.After(idleTime):
		i.invoke()
		return
	case <-ctx.Done():
		return
	}
}
