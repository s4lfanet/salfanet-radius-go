package eventbus

import (
	"log"
	"sync"
	"time"

	"github.com/s4lfanet/salfanet-radius-go/internal/tzutil"
)

// Event represents a system event that can trigger notifications.
type Event struct {
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

// EventHandler processes an event.
type EventHandler func(event Event) error

// EventBus is an in-memory pub/sub event system.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]EventHandler
	dedup       map[string]time.Time // dedup key → last fired time
	dedupMu     sync.Mutex
}

// New creates a new EventBus.
func New() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventHandler),
		dedup:       make(map[string]time.Time),
	}
}

// Subscribe registers a handler for a given event type.
func (b *EventBus) Subscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

// Publish emits an event to all subscribers. Non-blocking — handlers run in goroutines.
func (b *EventBus) Publish(eventType string, payload map[string]interface{}) {
	event := Event{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	b.mu.RLock()
	handlers, ok := b.subscribers[eventType]
	b.mu.RUnlock()

	if !ok {
		return
	}

	for _, handler := range handlers {
		go func(h EventHandler, e Event) {
			if err := h(e); err != nil {
				log.Printf("[EventBus] handler error for event %s: %v", e.Type, err)
			}
		}(handler, event)
	}
}

// PublishDedup emits an event but skips if the same dedupKey was fired within dedupWindow.
func (b *EventBus) PublishDedup(eventType, dedupKey string, dedupWindow time.Duration, payload map[string]interface{}) {
	b.dedupMu.Lock()
	if lastFired, ok := b.dedup[dedupKey]; ok {
		if time.Since(lastFired) < dedupWindow {
			b.dedupMu.Unlock()
			return // skip — already fired recently
		}
	}
	b.dedup[dedupKey] = time.Now()
	b.dedupMu.Unlock()

	b.Publish(eventType, payload)
}

// CleanupDedup removes expired dedup entries. Should be called periodically.
func (b *EventBus) CleanupDedup(maxAge time.Duration) {
	b.dedupMu.Lock()
	defer b.dedupMu.Unlock()
	now := time.Now()
	for key, firedAt := range b.dedup {
		if now.Sub(firedAt) > maxAge {
			delete(b.dedup, key)
		}
	}
}

// IsQuietHours checks if current time is within quiet hours (default 22:00-07:00 WIB).
func IsQuietHours(startHour, endHour int) bool {
	hour := tzutil.Now().Hour()
	if startHour <= endHour {
		return hour >= startHour && hour < endHour
	}
	// wraps midnight (e.g., 22-7)
	return hour >= startHour || hour < endHour
}
