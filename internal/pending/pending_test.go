package pending

import (
	"testing"
	"time"
)

func TestBufferAndRetrieve(t *testing.T) {
	mgr := NewManager(30 * time.Second)

	data := []byte("test stun packet")

	// Buffer a packet
	mgr.Buffer("ufrag1", data, "192.168.1.1:12345")

	// Verify it's buffered
	if !mgr.Has("ufrag1") {
		t.Error("Expected Has(ufrag1)=true")
	}

	// Retrieve it
	retrieved, ok := mgr.Retrieve("ufrag1")
	if !ok {
		t.Fatal("Expected to retrieve buffered packet")
	}

	// Verify data
	if string(retrieved) != string(data) {
		t.Errorf("Data mismatch: expected %s, got %s", string(data), string(retrieved))
	}

	// Verify it's removed
	if mgr.Has("ufrag1") {
		t.Error("Expected packet to be removed after retrieval")
	}

	// Verify count
	if mgr.Count() != 0 {
		t.Errorf("Expected count=0, got %d", mgr.Count())
	}
}

func TestCleanup(t *testing.T) {
	// Very short timeout for testing
	mgr := NewManager(10 * time.Millisecond)

	data := []byte("test")

	// Buffer a packet
	mgr.Buffer("ufrag1", data, "test")

	// Should exist immediately
	if !mgr.Has("ufrag1") {
		t.Fatal("Expected packet to exist")
	}

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// Cleanup
	expired := mgr.Cleanup()
	if len(expired) == 0 {
		t.Error("Expected expired packets")
	}
	if expired[0] != "ufrag1" {
		t.Errorf("Expected ufrag1 to expire, got %s", expired[0])
	}

	// Should be removed
	if mgr.Has("ufrag1") {
		t.Error("Expected packet to be removed after cleanup")
	}
}

func TestMultipleBuffers(t *testing.T) {
	mgr := NewManager(30 * time.Second)

	// Buffer multiple packets
	mgr.Buffer("ufrag1", []byte("data1"), "addr1")
	mgr.Buffer("ufrag2", []byte("data2"), "addr2")
	mgr.Buffer("ufrag3", []byte("data3"), "addr3")

	// Verify count
	if mgr.Count() != 3 {
		t.Errorf("Expected count=3, got %d", mgr.Count())
	}

	// Retrieve one
	mgr.Retrieve("ufrag2")

	// Verify count
	if mgr.Count() != 2 {
		t.Errorf("Expected count=2 after retrieval, got %d", mgr.Count())
	}

	// Verify remaining exist
	if !mgr.Has("ufrag1") || !mgr.Has("ufrag3") {
		t.Error("Expected remaining packets to exist")
	}

	// Verify retrieved one doesn't exist
	if mgr.Has("ufrag2") {
		t.Error("Expected ufrag2 to be removed")
	}
}