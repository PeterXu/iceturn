package stunparse

import (
	"testing"

	"github.com/pion/stun/v3"
)

func TestParseBindingRequest(t *testing.T) {
	// Create a simple STUN Binding Request
	msg, err := stun.Build(stun.BindingRequest, stun.TransactionID)
	if err != nil {
		t.Fatalf("Failed to build STUN message: %v", err)
	}

	// Add USERNAME attribute
	username := stun.Username("remote-ufrag:local-ufrag")
	msg.Add(username)

	// Encode the message
	err = msg.Encode()
	if err != nil {
		t.Fatalf("Failed to encode STUN message: %v", err)
	}

	// Parse the message
	result, err := ParseBindingRequest(msg.Raw)
	if err != nil {
		t.Fatalf("Failed to parse STUN Binding Request: %v", err)
	}

	// Verify results
	if result.RemoteUfrag != "remote-ufrag" {
		t.Errorf("Expected remote-ufrag='remote-ufrag', got '%s'", result.RemoteUfrag)
	}

	if result.LocalUfrag != "local-ufrag" {
		t.Errorf("Expected local-ufrag='local-ufrag', got '%s'", result.LocalUfrag)
	}

	t.Logf("Successfully parsed STUN Binding Request: remote=%s, local=%s",
		result.RemoteUfrag, result.LocalUfrag)
}

func TestParseBindingRequestInvalid(t *testing.T) {
	// Test with non-STUN data
	data := []byte("not a stun message")
	_, err := ParseBindingRequest(data)
	if err == nil {
		t.Error("Expected error for non-STUN data")
	}

	// Test with STUN message but not Binding Request
	msg, err := stun.Build(stun.NewType(stun.MethodAllocate, stun.ClassRequest), stun.TransactionID)
	if err != nil {
		t.Fatalf("Failed to build STUN message: %v", err)
	}
	err = msg.Encode()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	_, err = ParseBindingRequest(msg.Raw)
	if err == nil {
		t.Error("Expected error for non-Binding Request")
	}
}

func TestIsSTUNMessage(t *testing.T) {
	// Test with valid STUN message
	msg, err := stun.Build(stun.BindingRequest, stun.TransactionID)
	if err != nil {
		t.Fatalf("Failed to build STUN: %v", err)
	}
	err = msg.Encode()
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	if !IsSTUNMessage(msg.Raw) {
		t.Error("Expected IsSTUNMessage=true for valid STUN")
	}

	// Test with invalid data
	if IsSTUNMessage([]byte("not stun")) {
		t.Error("Expected IsSTUNMessage=false for invalid data")
	}
}