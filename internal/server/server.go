package server

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/peter/iceturn/internal/allocation"
	"github.com/peter/iceturn/internal/config"
	"github.com/peter/iceturn/internal/pending"
	"github.com/peter/iceturn/internal/stunparse"
	"github.com/pion/logging"
	"github.com/pion/turn/v5"
)

// IceTurnServer wraps pion/turn server with STUN interception
type IceTurnServer struct {
	config         *config.Config
	turnServer     *turn.Server
	allocManager   *allocation.Manager
	pendingManager *pending.Manager
	logger         logging.LeveledLogger

	// Track connections for allocation lookup
	connTracker *ConnectionTracker

	// Cleanup routine stop channel
	stopCleanup chan struct{}
}

// ConnectionTracker tracks UDP connections to allocations
type ConnectionTracker struct {
	mu            sync.RWMutex
	connToAlloc   map[string]*allocation.Allocation // srcAddr -> Allocation
	activeConns   map[string]net.Addr               // srcAddr -> remote addr
}

// NewConnectionTracker creates a new connection tracker
func NewConnectionTracker() *ConnectionTracker {
	return &ConnectionTracker{
		connToAlloc: make(map[string]*allocation.Allocation),
		activeConns: make(map[string]net.Addr),
	}
}

// TrackConnection records a connection for later lookup
func (t *ConnectionTracker) TrackConnection(srcAddr net.Addr, alloc *allocation.Allocation) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := srcAddr.String()
	t.connToAlloc[key] = alloc
	t.activeConns[key] = srcAddr
}

// GetAllocation retrieves allocation by source address
func (t *ConnectionTracker) GetAllocation(srcAddr net.Addr) (*allocation.Allocation, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	alloc, ok := t.connToAlloc[srcAddr.String()]
	return alloc, ok
}

// NewIceTurnServer creates a new IceTurn server
func NewIceTurnServer(cfg *config.Config) (*IceTurnServer, error) {
	loggerFactory := logging.NewDefaultLoggerFactory()
	logger := loggerFactory.NewLogger("iceturn")

	allocManager := allocation.NewManager()
	pendingManager := pending.NewManager(cfg.STUNPendingTimeout)
	connTracker := NewConnectionTracker()

	server := &IceTurnServer{
		config:         cfg,
		allocManager:   allocManager,
		pendingManager: pendingManager,
		connTracker:    connTracker,
		logger:         logger,
		stopCleanup:    make(chan struct{}),
	}

	// Start cleanup routine
	go pendingManager.StartCleanupRoutine(10 * time.Second)

	// Create underlying TURN server
	turnServer, err := server.createTURNServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create TURN server: %w", err)
	}

	server.turnServer = turnServer
	return server, nil
}

// createTURNServer creates the underlying pion/turn server with STUN interception
func (s *IceTurnServer) createTURNServer() (*turn.Server, error) {
	// Create UDP listener with STUN interception
	udpListener, err := net.ListenPacket("udp", s.config.ListenUDP)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP listener: %w", err)
	}

	// Wrap with STUN interception
	interceptedUDP := stunparse.NewSTUNInterceptConn(
		udpListener,
		s.handleSTUNPacket,
		s.logger,
	)

	// Create TCP listener (no STUN interception needed for TCP in this version)
	tcpListener, err := net.Listen("tcp", s.config.ListenTCP)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP listener: %w", err)
	}

	// Create TURN server
	turnServer, err := turn.NewServer(turn.ServerConfig{
		Realm: s.config.Realm,
		AuthHandler: func(ra *turn.RequestAttributes) (string, []byte, bool) {
			password, ok := s.config.Credentials[ra.Username]
			if !ok {
				return "", nil, false
			}
			return ra.Username, turn.GenerateAuthKey(ra.Username, ra.Realm, password), true
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: interceptedUDP,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(s.config.PublicIP),
					Address:      "0.0.0.0",
				},
			},
		},
		ListenerConfigs: []turn.ListenerConfig{
			{
				Listener: tcpListener,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(s.config.PublicIP),
					Address:      "0.0.0.0",
				},
			},
		},
		LoggerFactory: logging.NewDefaultLoggerFactory(),
	})

	return turnServer, err
}

// handleSTUNPacket processes intercepted STUN packets
func (s *IceTurnServer) handleSTUNPacket(data []byte, srcAddr net.Addr) ([]byte, error) {
	// Try to parse as STUN Binding Request
	result, err := stunparse.ParseBindingRequest(data)
	if err != nil {
		// Not a STUN Binding Request, pass through
		s.logger.Tracef("Not a STUN Binding Request from %s: %v", srcAddr, err)
		return data, nil
	}

	s.logger.Infof("STUN Binding Request from %s: local-ufrag=%s, remote-ufrag=%s",
		srcAddr, result.LocalUfrag, result.RemoteUfrag)

	// Lookup allocation for this connection
	alloc, ok := s.connTracker.GetAllocation(srcAddr)
	if !ok {
		s.logger.Warnf("No allocation found for %s, buffering STUN packet", srcAddr)
		// Buffer the packet even without allocation
		s.pendingManager.Buffer(result.LocalUfrag, data, srcAddr.String())
		return data, nil
	}

	// Register ice-ufrag mapping
	s.allocManager.RegisterIceUfrag(result.LocalUfrag, alloc)

	// Try to establish pair
	paired := s.allocManager.TryEstablishPair(result.LocalUfrag, result.RemoteUfrag)
	if paired {
		s.logger.Infof("Pair established: %s ↔ %s", result.LocalUfrag, result.RemoteUfrag)

		// Check if there's a pending STUN packet for the remote-ufrag
		if pendingData, ok := s.pendingManager.Retrieve(result.RemoteUfrag); ok {
			s.logger.Infof("Forwarding buffered STUN packet for %s", result.RemoteUfrag)
			// TODO: Forward pendingData to the paired allocation
			// This requires finding the paired allocation's connection
			// For now, we just log it
			_ = pendingData
		}

		// Return the STUN packet for normal processing
		return data, nil
	}

	// Pair not yet established, buffer this packet
	s.logger.Debugf("Buffering STUN packet for %s (waiting for pair)", result.LocalUfrag)
	s.pendingManager.Buffer(result.LocalUfrag, data, srcAddr.String())

	return data, nil
}

// Close shuts down the server
func (s *IceTurnServer) Close() error {
	close(s.stopCleanup)
	return s.turnServer.Close()
}

// AllocationCount returns the number of active allocations
func (s *IceTurnServer) AllocationCount() int {
	return s.turnServer.AllocationCount()
}