package stunparse

import (
	"fmt"
	"strings"

	"github.com/pion/stun/v3"
)

// ParseResult contains the parsed STUN Binding Request information
type ParseResult struct {
	RemoteUfrag string // remote ice-ufrag
	LocalUfrag  string // local ice-ufrag
	Message     *stun.Message
}

// ParseBindingRequest parses a STUN Binding Request to extract ice-ufrag
//
// STUN Binding Request format:
//   USERNAME: "remote-ufrag:local-ufrag"
//
// For example:
//   Client → Streamer: USERNAME: "streamer-ufrag:client-ufrag"
//   Streamer → Client: USERNAME: "client-ufrag:streamer-ufrag"
func ParseBindingRequest(data []byte) (*ParseResult, error) {
	// Check if it's a STUN message
	if !stun.IsMessage(data) {
		return nil, fmt.Errorf("not a STUN message")
	}

	// Decode STUN message
	msg := &stun.Message{
		Raw: make([]byte, len(data)),
	}
	copy(msg.Raw, data)

	if err := msg.Decode(); err != nil {
		return nil, fmt.Errorf("failed to decode STUN message: %w", err)
	}

	// Check if it's a Binding Request
	if msg.Type.Method != stun.MethodBinding || msg.Type.Class != stun.ClassRequest {
		return nil, fmt.Errorf("not a STUN Binding Request (method=%s, class=%s)",
			msg.Type.Method, msg.Type.Class)
	}

	// Extract USERNAME attribute
	var username stun.Username
	if err := username.GetFrom(msg); err != nil {
		return nil, fmt.Errorf("failed to get USERNAME attribute: %w", err)
	}

	// Parse ice-ufrag from username
	// Format: "remote-ufrag:local-ufrag"
	usernameStr := username.String()
	parts := strings.Split(usernameStr, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid USERNAME format: %s (expected 'remote-ufrag:local-ufrag')", usernameStr)
	}

	return &ParseResult{
		RemoteUfrag: parts[0],
		LocalUfrag:  parts[1],
		Message:     msg,
	}, nil
}

// IsSTUNMessage checks if data is a STUN message
func IsSTUNMessage(data []byte) bool {
	return stun.IsMessage(data)
}