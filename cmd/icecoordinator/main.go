package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/peter/iceturn/internal/allocation"
	"github.com/peter/iceturn/internal/config"
	"github.com/peter/iceturn/internal/pending"
	"github.com/peter/iceturn/internal/stunparse"
	"github.com/pion/logging"
	"github.com/pion/turn/v5"
)

func main() {
	// Load configuration
	cfg := config.DefaultConfig()

	// Initialize managers
	allocManager := allocation.NewManager()
	pendingManager := pending.NewManager(cfg.STUNPendingTimeout)

	// Start cleanup routine
	stopCleanup := pendingManager.StartCleanupRoutine(10 * time.Second)
	defer close(stopCleanup)

	// Create TURN server
	loggerFactory := logging.NewDefaultLoggerFactory()
	logger := loggerFactory.NewLogger("iceturn")

	server, err := turn.NewServer(turn.ServerConfig{
		Realm: cfg.Realm,
		AuthHandler: func(username string, realm string, srcAddr net.Addr) (string, []byte, bool) {
			// Simple authentication for demo
			// TODO: Implement proper credential validation
			password, ok := cfg.Credentials[username]
			if !ok {
				return "", nil, false
			}
			return username, turn.GenerateAuthKey(username, realm, password), true
		},
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn: createUDPListener(cfg.ListenUDP, logger),
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(cfg.PublicIP),
					Address:      "0.0.0.0",
				},
			},
		},
		ListenerConfigs: []turn.ListenerConfig{
			{
				Listener: createTCPListener(cfg.ListenTCP, logger),
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
					RelayAddress: net.ParseIP(cfg.PublicIP),
					Address:      "0.0.0.0",
				},
			},
		},
		LoggerFactory: loggerFactory,
	})
	if err != nil {
		log.Fatalf("Failed to create TURN server: %v", err)
	}
	defer server.Close()

	// TODO: Intercept STUN packets for ice-ufrag pairing
	// This requires modifying the underlying pion/turn server or
	// creating a custom packet handler

	logger.Infof("IceTurn server started on %s (UDP) and %s (TCP)", cfg.ListenUDP, cfg.ListenTCP)
	logger.Infof("Public IP: %s", cfg.PublicIP)
	logger.Infof("STUN pending timeout: %v", cfg.STUNPendingTimeout)

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down IceTurn server...")
}

// createUDPListener creates a UDP listener
func createUDPListener(addr string, logger logging.LeveledLogger) net.PacketConn {
	conn, err := net.ListenPacket("udp", addr)
	if err != nil {
		logger.Errorf("Failed to create UDP listener: %v", err)
		os.Exit(1)
	}
	return conn
}

// createTCPListener creates a TCP listener
func createTCPListener(addr string, logger logging.LeveledLogger) net.Listener {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Errorf("Failed to create TCP listener: %v", err)
		os.Exit(1)
	}
	return listener
}

// handleSTUNPacket intercepts and handles STUN packets for ice-ufrag pairing
// TODO: Integrate this into the packet flow
func handleSTUNPacket(
	data []byte,
	srcAddr net.Addr,
	allocManager *allocation.Manager,
	pendingManager *pending.Manager,
	logger logging.LeveledLogger,
) ([]byte, error) {
	// Check if it's a STUN message
	if !stunparse.IsSTUNMessage(data) {
		// Not STUN, forward as-is
		return data, nil
	}

	// Try to parse as STUN Binding Request
	result, err := stunparse.ParseBindingRequest(data)
	if err != nil {
		// Not a Binding Request, forward as-is
		logger.Debugf("Not a STUN Binding Request: %v", err)
		return data, nil
	}

	logger.Debugf("STUN Binding Request: local-ufrag=%s, remote-ufrag=%s",
		result.LocalUfrag, result.RemoteUfrag)

	// Register local ice-ufrag to allocation mapping
	// Note: We need to find the allocation corresponding to srcAddr
	// This is a placeholder - actual implementation needs allocation lookup
	// allocManager.RegisterIceUfrag(result.LocalUfrag, alloc)

	// Try to establish pair
	paired := allocManager.TryEstablishPair(result.LocalUfrag, result.RemoteUfrag)
	if paired {
		logger.Infof("Pair established: %s ↔ %s", result.LocalUfrag, result.RemoteUfrag)

		// Check if there's a pending STUN packet for the remote-ufrag
		if pendingData, ok := pendingManager.Retrieve(result.RemoteUfrag); ok {
			logger.Debugf("Forwarding buffered STUN packet for %s", result.RemoteUfrag)
			// TODO: Forward pendingData to the paired allocation
			// This requires access to the allocation's connection
			_ = pendingData
		}
	} else {
		// Pair not yet established, buffer this packet
		logger.Debugf("Buffering STUN packet for %s (waiting for pair)", result.LocalUfrag)
		pendingManager.Buffer(result.LocalUfrag, data, srcAddr.String())
	}

	return data, nil
}