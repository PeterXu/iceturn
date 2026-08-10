package allocation

import (
	"sync"
	"time"
)

// Allocation represents a TURN allocation (simplified for our use case)
type Allocation struct {
	ID        string
	Conn      interface{} // Connection associated with this allocation
	PeerAlloc *Allocation  // Paired allocation
}

// PairState represents the state of an allocation pair
type PairState struct {
	LocalUfrag  string
	RemoteUfrag string
	LocalAlloc  *Allocation
	RemoteAlloc *Allocation
	Paired      bool
	CreatedAt   time.Time
	PairedAt    time.Time
}

// Manager manages ice-ufrag to allocation mappings and pairings
type Manager struct {
	mu sync.RWMutex

	// ice-ufrag → Allocation mapping
	iceUfragMap map[string]*Allocation

	// ice-ufrag → PairState (tracks pairing progress)
	pairMap map[string]*PairState

	// Allocation → ice-ufrag reverse mapping
	allocToUfrag map[*Allocation]string
}

// NewManager creates a new allocation manager
func NewManager() *Manager {
	return &Manager{
		iceUfragMap:  make(map[string]*allocation.Allocation),
		pairMap:      make(map[string]*PairState),
		allocToUfrag: make(map[*allocation.Allocation]string),
	}
}

// RegisterIceUfrag registers an ice-ufrag to allocation mapping
func (m *Manager) RegisterIceUfrag(ufrag string, alloc *Allocation) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.iceUfragMap[ufrag] = alloc
	m.allocToUfrag[alloc] = ufrag
}

// GetAllocation retrieves allocation by ice-ufrag
func (m *Manager) GetAllocation(ufrag string) (*Allocation, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	alloc, ok := m.iceUfragMap[ufrag]
	return alloc, ok
}

// GetIceUfrag retrieves ice-ufrag by allocation
func (m *Manager) GetIceUfrag(alloc *Allocation) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ufrag, ok := m.allocToUfrag[alloc]
	return ufrag, ok
}

// TryEstablishPair attempts to establish a pairing between two ice-ufrags
// Returns true if pair was successfully established
func (m *Manager) TryEstablishPair(localUfrag, remoteUfrag string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	localAlloc, localOk := m.iceUfragMap[localUfrag]
	remoteAlloc, remoteOk := m.iceUfragMap[remoteUfrag]

	if !localOk || !remoteOk {
		// One or both allocations not registered yet
		// Record the attempt for later
		if _, exists := m.pairMap[localUfrag]; !exists {
			m.pairMap[localUfrag] = &PairState{
				LocalUfrag:  localUfrag,
				RemoteUfrag: remoteUfrag,
				LocalAlloc:  localAlloc,
				RemoteAlloc: remoteAlloc,
				CreatedAt:   time.Now(),
			}
		}
		return false
	}

	// Both allocations exist, establish pair
	pair := &PairState{
		LocalUfrag:  localUfrag,
		RemoteUfrag: remoteUfrag,
		LocalAlloc:  localAlloc,
		RemoteAlloc: remoteAlloc,
		Paired:      true,
		CreatedAt:   time.Now(),
		PairedAt:    time.Now(),
	}

	m.pairMap[localUfrag] = pair
	m.pairMap[remoteUfrag] = pair

	return true
}

// GetPeerAllocation retrieves the paired allocation for a given ice-ufrag
func (m *Manager) GetPeerAllocation(ufrag string) (*Allocation, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pair, ok := m.pairMap[ufrag]
	if !ok || !pair.Paired {
		return nil, false
	}

	// Return the peer allocation
	if pair.LocalUfrag == ufrag {
		return pair.RemoteAlloc, true
	}
	return pair.LocalAlloc, true
}

// Cleanup removes expired pairings (older than timeout)
func (m *Manager) Cleanup(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for ufrag, pair := range m.pairMap {
		if !pair.Paired && now.Sub(pair.CreatedAt) > timeout {
			// Remove unpaired expired entries
			delete(m.pairMap, ufrag)
		}
	}
}