package allocation

import (
	"testing"
	"time"
)

func TestRegisterAndGet(t *testing.T) {
	mgr := NewManager()

	alloc1 := &Allocation{ID: "alloc1"}
	alloc2 := &Allocation{ID: "alloc2"}

	// Register ice-ufrags
	mgr.RegisterIceUfrag("ufrag1", alloc1)
	mgr.RegisterIceUfrag("ufrag2", alloc2)

	// Test GetAllocation
	retrieved1, ok := mgr.GetAllocation("ufrag1")
	if !ok {
		t.Fatal("Expected to find allocation for ufrag1")
	}
	if retrieved1 != alloc1 {
		t.Error("Wrong allocation retrieved")
	}

	retrieved2, ok := mgr.GetAllocation("ufrag2")
	if !ok {
		t.Fatal("Expected to find allocation for ufrag2")
	}
	if retrieved2 != alloc2 {
		t.Error("Wrong allocation retrieved")
	}

	// Test GetIceUfrag (reverse lookup)
	ufrag1, ok := mgr.GetIceUfrag(alloc1)
	if !ok {
		t.Fatal("Expected to find ufrag for alloc1")
	}
	if ufrag1 != "ufrag1" {
		t.Errorf("Expected ufrag1, got %s", ufrag1)
	}

	// Test non-existent
	_, ok = mgr.GetAllocation("nonexistent")
	if ok {
		t.Error("Expected not to find non-existent ufrag")
	}
}

func TestPairEstablishment(t *testing.T) {
	mgr := NewManager()

	alloc1 := &Allocation{ID: "alloc1"}
	alloc2 := &Allocation{ID: "alloc2"}

	// Register both
	mgr.RegisterIceUfrag("client-ufrag", alloc1)
	mgr.RegisterIceUfrag("streamer-ufrag", alloc2)

	// Try to establish pair (should succeed)
	paired := mgr.TryEstablishPair("client-ufrag", "streamer-ufrag")
	if !paired {
		t.Fatal("Expected pairing to succeed")
	}

	// Verify pairing
	peer1, ok := mgr.GetPeerAllocation("client-ufrag")
	if !ok {
		t.Fatal("Expected to find peer for client-ufrag")
	}
	if peer1 != alloc2 {
		t.Error("Wrong peer for client")
	}

	peer2, ok := mgr.GetPeerAllocation("streamer-ufrag")
	if !ok {
		t.Fatal("Expected to find peer for streamer-ufrag")
	}
	if peer2 != alloc1 {
		t.Error("Wrong peer for streamer")
	}

	t.Log("Pair established successfully")
}

func TestPairEstablishmentIncomplete(t *testing.T) {
	mgr := NewManager()

	alloc1 := &Allocation{ID: "alloc1"}
	mgr.RegisterIceUfrag("ufrag1", alloc1)

	// Try to pair with non-existent ufrag2
	paired := mgr.TryEstablishPair("ufrag1", "ufrag2")
	if paired {
		t.Error("Expected pairing to fail when remote not registered")
	}

	// Later, register ufrag2
	alloc2 := &Allocation{ID: "alloc2"}
	mgr.RegisterIceUfrag("ufrag2", alloc2)

	// Now pairing should succeed when we retry
	paired = mgr.TryEstablishPair("ufrag1", "ufrag2")
	if !paired {
		t.Fatal("Expected pairing to succeed after both registered")
	}

	t.Log("Delayed pairing succeeded")
}

func TestCleanup(t *testing.T) {
	mgr := NewManager()

	alloc1 := &Allocation{ID: "alloc1"}
	mgr.RegisterIceUfrag("ufrag1", alloc1)

	// Try to pair with non-existent (creates pending pair)
	mgr.TryEstablishPair("ufrag1", "ufrag2")

	// Cleanup with very short timeout (should remove pending pair)
	mgr.Cleanup(1 * time.Nanosecond)

	// Verify pending pair was removed
	paired := mgr.TryEstablishPair("ufrag1", "ufrag2")
	if paired {
		t.Error("Expected pending pair to be cleaned up")
	}
}