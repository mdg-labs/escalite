package realtime

import (
	"context"
	"log/slog"
	"sync"
)

// Bridge owns the in-process hub and Postgres LISTEN worker.
type Bridge struct {
	Hub      *Hub
	listener *Listener

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// NewBridge wires a hub and listener for database NOTIFY fan-out.
func NewBridge(connString string, logger *slog.Logger) *Bridge {
	hub := NewHub()
	return &Bridge{
		Hub:      hub,
		listener: NewListener(connString, hub, logger),
	}
}

// Start launches the listener in a background goroutine.
func (b *Bridge) Start(ctx context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cancel != nil {
		return
	}

	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	b.cancel = cancel
	b.done = done

	go func() {
		defer close(done)
		b.listener.Run(runCtx)
	}()
}

// Close stops the listener and waits for shutdown.
func (b *Bridge) Close() {
	b.mu.Lock()
	cancel := b.cancel
	done := b.done
	b.cancel = nil
	b.done = nil
	b.mu.Unlock()

	if cancel != nil {
		cancel()
		<-done
	}
}
