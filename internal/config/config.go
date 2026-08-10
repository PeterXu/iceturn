package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
		PublicIP:           "127.0.0.1",
		STUNPendingTimeout: 30 * time.Second,
		Credentials:        make(map[string]string),
	}
}

// LoadFromEnv loads configuration from environment variables
// Environment variables override config file values
func (c *Config) LoadFromEnv() {
	if realm := os.Getenv("REALM"); realm != "" {
		c.Realm = realm
	}

	if publicIP := os.Getenv("PUBLIC_IP"); publicIP != "" {
		c.PublicIP = publicIP
	}

	if listenUDP := os.Getenv("LISTEN_UDP"); listenUDP != "" {
		c.ListenUDP = listenUDP
	}

	if listenTCP := os.Getenv("LISTEN_TCP"); listenTCP != "" {
		c.ListenTCP = listenTCP
	}

	if timeout := os.Getenv("STUN_PENDING_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			c.STUNPendingTimeout = duration
		}
	}

	// Load credentials from environment
	// Format: TURN_CREDENTIALS=user1:pass1,user2:pass2
	if users := os.Getenv("TURN_CREDENTIALS"); users != "" {
		c.parseCredentials(users)
	}
}

// parseCredentials parses credentials from format: user1:pass1,user2:pass2
func (c *Config) parseCredentials(creds string) {
	pairs := strings.Split(creds, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, ":")
		if len(parts) == 2 {
			c.Credentials[parts[0]] = parts[1]
		}
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Realm == "" {
		return fmt.Errorf("realm cannot be empty")
	}

	if c.PublicIP == "" {
		return fmt.Errorf("public_ip cannot be empty")
	}

	if c.STUNPendingTimeout < 1*time.Second {
		return fmt.Errorf("stun_pending_timeout must be at least 1 second")
	}

	return nil
}

// GetPort extracts port number from address string
func (c *Config) GetPort(addr string) int {
	// Extract port from ":3478" or "0.0.0.0:3478"
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			port, _ := strconv.Atoi(addr[i+1:])
			return port
		}
	}
	return 0
}

// String returns a string representation of the config
func (c *Config) String() string {
	return fmt.Sprintf("Config{Realm=%s, PublicIP=%s, ListenUDP=%s, ListenTCP=%s, Timeout=%v, Credentials=%d}",
		c.Realm, c.PublicIP, c.ListenUDP, c.ListenTCP, c.STUNPendingTimeout, len(c.Credentials))
}
