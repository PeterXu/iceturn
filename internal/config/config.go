package config

import (
	"time"
)

// Config represents the IceTurn server configuration
type Config struct {
	// Realm for TURN authentication
	Realm string `yaml:"realm"`

	// Listener addresses
	ListenUDP string `yaml:"listen_udp"`
	ListenTCP string `yaml:"listen_tcp"`

	// Public IP for relay address
	PublicIP string `yaml:"public_ip"`

	// STUN packet buffering timeout
	STUNPendingTimeout time.Duration `yaml:"stun_pending_timeout"`

	// Authentication credentials
	Credentials map[string]string `yaml:"credentials"` // username -> password
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Realm:              "iceturn.local",
		ListenUDP:          ":3478",
		ListenTCP:          ":3478",
		PublicIP:           "0.0.0.0",
		STUNPendingTimeout: 30 * time.Second,
		Credentials:        make(map[string]string),
	}
}