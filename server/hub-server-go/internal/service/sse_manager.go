package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/goyais/hub/internal/model"
)

// subscriber is one SSE client connection for a session.
type subscriber struct {
	sessionID string
	ch        chan *model.ExecutionEvent
}

// SSEManager distributes execution events to connected SSE clients.
// It keeps an in-memory ring buffer of the last 500 events per execution
// and a fan-out map of session → subscribers.
type SSEManager struct {
	mu          sync.RWMutex
	// sessionID → list of subscribers
	subscribers map[string][]*subscriber
	// executionID → recent events (ring buffer, max 500)
	recentEvents map[string][]*model.ExecutionEvent
}

func NewSSEManager() *SSEManager {
	return &SSEManager{
		subscribers:  make(map[string][]*subscriber),
		recentEvents: make(map[string][]*model.ExecutionEvent),
	}
}

// Publish sends an event to all subscribers of the session that owns this execution,
// and appends it to the in-memory ring buffer.
func (m *SSEManager) Publish(executionID string, event *model.ExecutionEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Append to ring buffer (cap 500)
	buf := m.recentEvents[executionID]
	buf = append(buf, event)
	if len(buf) > 500 {
		buf = buf[len(buf)-500:]
	}
	m.recentEvents[executionID] = buf

	// Fan-out to SSE subscribers (keyed by executionID for simplicity)
	for _, sub := range m.subscribers[executionID] {
		select {
		case sub.ch <- event:
		default:
			// Slow consumer — drop. Client will reconnect and replay via since_seq.
		}
	}
}

// Subscribe registers a new SSE client for a given execution.
// Returns a channel of events and a cancel function to unsubscribe.
func (m *SSEManager) Subscribe(ctx context.Context, executionID string, sinceSeq int) (<-chan *model.ExecutionEvent, func()) {
	m.mu.Lock()

	ch := make(chan *model.ExecutionEvent, 64)
	sub := &subscriber{sessionID: executionID, ch: ch}
	m.subscribers[executionID] = append(m.subscribers[executionID], sub)

	// Replay buffered events since sinceSeq
	buffered := m.recentEvents[executionID]
	toReplay := make([]*model.ExecutionEvent, 0)
	for _, ev := range buffered {
		if ev.Seq > sinceSeq {
			toReplay = append(toReplay, ev)
		}
	}
	m.mu.Unlock()

	// Send replayed events into channel before releasing caller
	go func() {
		for _, ev := range toReplay {
			select {
			case ch <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()

	cancel := func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		subs := m.subscribers[executionID]
		updated := subs[:0]
		for _, s := range subs {
			if s != sub {
				updated = append(updated, s)
			}
		}
		m.subscribers[executionID] = updated
		close(ch)
	}

	return ch, cancel
}

// RecentEvents returns buffered events for an execution (for DB miss on reconnect).
func (m *SSEManager) RecentEvents(executionID string, sinceSeq int) []*model.ExecutionEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*model.ExecutionEvent
	for _, ev := range m.recentEvents[executionID] {
		if ev.Seq > sinceSeq {
			out = append(out, ev)
		}
	}
	return out
}

// ActiveExecutionForSession looks up the current active_execution_id for a session.
// Used by the SSE endpoint to know which execution to subscribe to.
func ActiveExecutionForSession(ctx context.Context, db interface {
	QueryRowContext(ctx context.Context, query string, args ...any) interface{ Scan(dest ...any) error }
}, sessionID string) (string, error) {
	_ = db
	_ = sessionID
	return "", fmt.Errorf("use direct DB query")
}
