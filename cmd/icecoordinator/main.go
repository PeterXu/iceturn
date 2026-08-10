package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/peter/iceturn/internal/config"
	"github.com/peter/iceturn/internal/server"
)

func main() {
	// Load configuration
	cfg := config.DefaultConfig()

	// Create IceTurn server
	iceTurnServer, err := server.NewIceTurnServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create IceTurn server: %v", err)
	}
	defer iceTurnServer.Close()

	log.Printf("IceTurn server started on %s (UDP) and %s (TCP)", cfg.ListenUDP, cfg.ListenTCP)
	log.Printf("Public IP: %s", cfg.PublicIP)
	log.Printf("STUN pending timeout: %v", cfg.STUNPendingTimeout)
	log.Println("Ready to accept connections...")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down IceTurn server...")
}