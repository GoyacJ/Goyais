package eventbus

import (
	"context"
	"fmt"
	"sync"

	"goyais/services/hub/internal/domain"
)

type Subscription struct {
	ch     chan domain.RunEvent
	once   sync.Once
	close  func()
	closed bool
}

func (s *Subscription) Events() <-chan domain.RunEvent {
	if s == nil {
		return nil
	}
	return s.ch
}

func (s *Subscription) Close() error {
	if s == nil {
		return nil
	}
	s.once.Do(func() {
		if s.close != nil {
			s.close()
		}
		s.closed = true
	})
	return nil
}

type MemoryBus struct {
	mu          sync.RWMutex
	bufferSize  int
	subscribers map[*Subscription]struct{}
}

func NewMemoryBus(bufferSize int) *MemoryBus {
	if bufferSize <= 0 {
		bufferSize = 1
	}
	return &MemoryBus{
		bufferSize:  bufferSize,
		subscribers: map[*Subscription]struct{}{},
	}
}

func (b *MemoryBus) Subscribe(_ context.Context) (domain.EventSubscription, error) {
	subscription := &Subscription{
		ch: make(chan domain.RunEvent, b.bufferSize),
	}
	subscription.close = func() {
		b.mu.Lock()
		delete(b.subscribers, subscription)
		close(subscription.ch)
		b.mu.Unlock()
	}

	b.mu.Lock()
	b.subscribers[subscription] = struct{}{}
	b.mu.Unlock()
	return subscription, nil
}

func (b *MemoryBus) Publish(_ context.Context, event domain.RunEvent) error {
	if b == nil {
		return fmt.Errorf("publish event: bus is nil")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	for subscription := range b.subscribers {
		select {
		case subscription.ch <- event:
		default:
			<-subscription.ch
			subscription.ch <- event
		}
	}
	return nil
}

var _ domain.EventBus = (*MemoryBus)(nil)
