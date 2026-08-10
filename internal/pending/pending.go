package pending

import (
	"sync"
	"time"
)

// PendingSTUN represents a buffered STUN packet
type PendingSTUN struct {
	Ufrag     string
	Data      []byte
	Timestamp time.Time
	Source    string // source address (for logging/debugging)
}

// Manager manages pending (buffered) STUN packets
type Manager struct {
	mu sync.RWMutex

	// local-ufrag → PendingSTUN
	pending map[string]*PendingSTUN

	// timeout for pending packets
	timeout time.Duration
}

// NewManager creates a new pending STUN packet manager
func NewManager(timeout time.Duration) *Manager {
	return &Manager{
		pending: make(map[string]*PendingSTUN),
		timeout: timeout,
	}
}

// Buffer stores a STUN packet for later forwarding
func (m *Manager) Buffer(ufrag string, data []byte, source string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Make a copy of data
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	m.pending[ufrag] = &PendingSTUN{
		Ufrag:     ufrag,
		Data:      dataCopy,
		Timestamp: time.Now(),
		Source:    source,
	}
}

// Retrieve gets a buffered STUN packet and removes it from pending
func (m *Manager) Retrieve(ufrag string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	pending, ok := m.pending[ufrag]
	if !ok {
		return nil, false
	}

	// Remove from pending
	delete(m.pending, ufrag)

	return pending.Data, true
}

// Has checks if there's a pending packet for a given ufrag
func (m *Manager) Has(ufrag string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.pending[ufrag]
	return ok
}

// Cleanup removes expired pending packets
func (m *Manager) Cleanup() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var expired []string
	now := time.Now()

	for ufrag, pending := range m.pending {
		if now.Sub(pending.Timestamp) > m.timeout {
			delete(m.pending, ufrag)
			expired = append(expired, ufrag)
		}
	}

	return expired
}

// Count returns the number of pending packets
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.pending)
}

// StartCleanupRoutine starts a periodic cleanup routine
func (m *Manager) StartCleanupRoutine(interval time.Duration) chan struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.Cleanup()
			case <-stop:
				return
			}
		}
	}()

	return stop
}