# Custom TURN Server Design

> Status: Draft
> Date: 2026-08-10

## Goal

Replace coturn with a custom TURN server using pion/turn, supporting single-port multiplexing with ICE-ufrag pairing for deterministic WebRTC connections.

## Problem

- Current: Each TURN allocation uses random ports → Client must complete full ICE negotiation
- Goal: Single port with ice-ufrag pairing → Client can directly connect without ICE discovery
- Multi-client/streamer support on same port

---

## Solution Overview: Method C (Recommended)

### Core Design

```
Streamer (browser)
  │
  ├─ TURN Allocate (standard credential)
  │
  └─ STUN Binding Request (ice-ufrag)
      → TURN Server parses and pairs

TURN Server (pion/turn modified)
  │
  ├─ Standard TURN Allocate (uses TURN credential)
  ├─ Parse STUN Binding Request (extract ice-ufrag)
  ├─ Establish ice-ufrag → Allocation mapping
  ├─ Buffer first STUN packet until pair established
  └─ Direct forwarding based on pairing

Client (browser)
  │
  ├─ TURN Allocate (standard credential)
  └─ STUN Binding Request (ice-ufrag)
      → TURN Server parses and pairs
```

---

## Key Insight

**ICE Full Mode Behavior:**
- Both client and streamer send STUN Binding Request
- STUN Binding Request contains: `USERNAME: "remote-ufrag:local-ufrag"`
- TURN Server can parse STUN packets to establish pairing
- No need for different relay addresses to distinguish connections

---

## Complete Flow

```
Streamer                         TURN Server                          Client
   │                                 │                                    │
   │ 1. Allocate Request            │                                    │
   │    USERNAME: "turn-user-s"     │                                    │
   │    (standard TURN credential)  │                                    │
   ├────────────────────────────────>│                                    │
   │                                 │ Record: turn-user-s → Allocation-A │
   │                                 │                                    │
   │ 2. Allocate Success            │                                    │
   │    RELAYED: 192.168.1.1:3478   │                                    │
   │<────────────────────────────────┤                                    │
   │                                 │                                    │
   │                                 │         3. Allocate Request        │
   │                                 │            USERNAME: "turn-user-c" │
   │                                 │<───────────────────────────────────┤
   │                                 │                                    │
   │                                 │         Record: turn-user-c → Allocation-B
   │                                 │                                    │
   │                                 │         4. Allocate Success        │
   │                                 │         RELAYED: 192.168.1.1:3478 │
   │                                 ├───────────────────────────────────>│
   │                                 │                                    │
   │                                 │                                    │
   │                                 │         5. STUN Binding Request    │
   │                                 │            USERNAME: "streamer-ufrag:client-ufrag"
   │                                 │            (ice-ufrag)             │
   │                                 │<───────────────────────────────────┤
   │                                 │                                    │
   │                                 │ Parse STUN:                        │
   │                                 │ - ice-ufrag: client-ufrag          │
   │                                 │ - Allocation: Allocation-B         │
   │                                 │                                    │
   │                                 │ Record: client-ufrag → Allocation-B│
   │                                 │                                    │
   │                                 │ Buffer STUN packet (waiting pair)  │
   │                                 │                                    │
   │                                 │                                    │
   │ 6. STUN Binding Request        │                                    │
   │    USERNAME: "client-ufrag:streamer-ufrag"                          │
   │    (ice-ufrag)                  │                                    │
   ├────────────────────────────────>│                                    │
   │                                 │                                    │
   │                                 │ Parse STUN:                        │
   │                                 │ - ice-ufrag: streamer-ufrag        │
   │                                 │ - Allocation: Allocation-A         │
   │                                 │                                    │
   │                                 │ Record: streamer-ufrag → Allocation-A│
   │                                 │                                    │
   │                                 │ Establish pair: Allocation-A ↔ Allocation-B│
   │                                 │                                    │
   │                                 │ Forward buffered STUN packet (step 5)│
   │                                 │                                    │
   │ 7. STUN Binding Request        │                                    │
   │    (forwarded from step 5)      │                                    │
   │<────────────────────────────────┤                                    │
   │                                 │                                    │
   │                                 │                                    │
   │ 8. STUN Binding Response       │                                    │
   ├────────────────────────────────>│                                    │
   │                                 │ Lookup Allocation-A → pair Allocation-B │
   │                                 │                                    │
   │                                 │ Forward to Allocation-B            │
   │                                 ├───────────────────────────────────>│
   │                                 │                                    │
   │                                 │         9. STUN Binding Response   │
   │                                 │<───────────────────────────────────┤
   │                                 │                                    │
   │                                 │ Lookup Allocation-B → pair Allocation-A │
   │                                 │                                    │
   │ 10. STUN Binding Response      │                                    │
   │<────────────────────────────────┤                                    │
   │                                 │                                    │
   │                                 │                                    │
   │ 11. DTLS/SRTP data             │                                    │
   ├────────────────────────────────>│ Lookup pair, forward               │
   │                                 ├───────────────────────────────────>│
```

---

## Key Mechanisms

### 1. Allocate uses standard TURN credential

- Allocate Request USERNAME is TURN credential (for authentication)
- Not using ice-ufrag as TURN username
- Fully compliant with TURN standard

### 2. STUN Binding contains ice-ufrag

- STUN Binding Request USERNAME format: `"remote-ufrag:local-ufrag"`
- This is standard ICE protocol behavior
- Both sides send (ICE Full mode)

### 3. TURN Server parses STUN to build ice-ufrag mapping

```go
// Parse STUN Binding Request
func parseSTUNBinding(msg *stun.Message) (remoteUfrag, localUfrag string) {
    username := extractUsername(msg)  // "streamer-ufrag:client-ufrag"
    parts := strings.Split(username, ":")
    return parts[0], parts[1]  // remote, local
}

// Build ice-ufrag → Allocation mapping
func (s *TURNServer) handleSTUNBinding(conn *net.UDPConn, msg *stun.Message) {
    remoteUfrag, localUfrag := parseSTUNBinding(msg)

    // Find sender's allocation
    allocation := s.findAllocationByConn(conn)

    // Record ice-ufrag mapping
    s.iceUfragMap[localUfrag] = allocation

    // Buffer STUN packet
    s.pendingSTUN[localUfrag] = msg
}
```

### 4. Establish pair when both sides send

```go
func (s *TURNServer) tryEstablishPair(localUfrag, remoteUfrag string) {
    // Check if both ice-ufrag are mapped
    localAlloc := s.iceUfragMap[localUfrag]
    remoteAlloc := s.iceUfragMap[remoteUfrag]

    if localAlloc != nil && remoteAlloc != nil {
        // Establish pair
        localAlloc.peerAllocation = remoteAlloc
        remoteAlloc.peerAllocation = localAlloc

        // Forward buffered STUN packets
        s.forwardPendingSTUN(localUfrag, remoteUfrag)
    }
}
```

### 5. Data forwarding based on pairing

```go
func (s *TURNServer) handleDataPacket(conn *net.UDPConn, data []byte) {
    allocation := s.findAllocationByConn(conn)

    if allocation.peerAllocation != nil {
        // Forward directly to paired allocation
        allocation.peerAllocation.conn.WriteTo(data, allocation.peerAllocation.peerAddr)
    }
}
```

---

## Multi Client-Streamer Support

**Scenario:**
```
Streamer1: Allocation-A (ice-ufrag: s1)
Streamer2: Allocation-C (ice-ufrag: s2)

Client1: Allocation-B (ice-ufrag: c1)
Client2: Allocation-D (ice-ufrag: c2)
```

**Pair establishment:**
```
Client1 → TURN: STUN Binding (USERNAME: "s1:c1")
Streamer1 → TURN: STUN Binding (USERNAME: "c1:s1")
TURN establishes: Allocation-A ↔ Allocation-B

Client2 → TURN: STUN Binding (USERNAME: "s2:c2")
Streamer2 → TURN: STUN Binding (USERNAME: "c2:s2")
TURN establishes: Allocation-C ↔ Allocation-D
```

**Data forwarding:**
- Single port 3478 supports multiple client-streamer pairs
- Distinguished by ice-ufrag pairing relationship
- No need for different relay addresses

---

## STUN Packet Buffering

**Timing issue:**
- Client sends STUN Binding Request first
- Streamer may not be connected or sent STUN yet
- TURN Server needs to buffer first STUN packet

**Handling strategy:**
```
1. Receive first STUN packet
   - Record ice-ufrag mapping
   - Buffer STUN packet (with timeout, e.g., 30 seconds)

2. Receive paired STUN packet
   - Establish pair relationship
   - Forward previously buffered STUN packet

3. Timeout handling
   - If paired STUN not received within 30 seconds
   - Delete buffered STUN packet
```

---

## ICE Full Mode Necessity

**ICE Full Mode:**
- Both sides are ICE agents
- Both sides initiate connectivity checks
- Both sides send STUN Binding Request

**Browser support:**
- Chrome: ICE Full mode (default)
- Firefox: ICE Full mode (default)
- Safari: ICE Full mode (default)

**ICE Lite mode (not applicable):**
- Only one side sends STUN Binding Request
- Typically used for server-side (VoIP servers)
- Browsers generally don't use ICE Lite

**Verification:**
```
Browsers default to ICE Full mode
↓
Both sides send STUN Binding Request
↓
TURN Server automatically parses and establishes pairs
↓
No browser modifications needed
```

---

## Implementation Requirements

### Modify pion/turn Server

1. **Parse STUN Binding Request**
   - Add STUN message parsing logic
   - Extract USERNAME attribute
   - Parse ice-ufrag

2. **Build ice-ufrag mapping**
   - Maintain `iceUfragMap: map[string]*Allocation`
   - Record ice-ufrag → Allocation mapping

3. **Buffer STUN packets**
   - Maintain `pendingSTUN: map[string]*stun.Message`
   - Set timeout mechanism (30 seconds)

4. **Pairing logic**
   - Detect both sides sent STUN Binding
   - Establish pair relationship: `Allocation.peerAllocation`

5. **Data forwarding**
   - Forward data based on pair relationship
   - Don't rely on relay address for distinction

### Configuration Example

```yaml
# config.yaml
realm: magpie.local

# Single port listening
listen_udp: 3478
listen_tcp: 3478

# No port pool needed (single-port multiplexing)
# No fixed port allocation needed

# STUN packet buffering timeout
stun_pending_timeout: 30s
```

---

## Advantages Summary

**Compared to Method A/B:**
- ✅ No port pool management
- ✅ No REST API to query ports
- ✅ Automatic pairing, no extra signaling
- ✅ Supports unlimited client-streamer pairs (memory-limited only)

**Standard compliance:**
- ✅ Standard TURN Allocate (uses TURN credential)
- ✅ Standard ICE Full mode (browser default)
- ✅ Standard STUN Binding Request (contains ice-ufrag)

**Implementation complexity:**
- ⚠️ Requires modifying TURN Server to parse STUN packets
- ⚠️ Requires implementing buffering and pairing logic
- ✅ No browser-side modifications needed

---

## Next Steps

- [x] Design Method C: single-port multiplexing + ice-ufrag pairing
- [ ] Implement core TURN server modifications
  - [ ] STUN Binding Request parsing
  - [ ] ice-ufrag mapping management
  - [ ] STUN packet buffering mechanism
  - [ ] Pairing logic
  - [ ] Data forwarding based on pairing
- [ ] Integration testing
  - [ ] Single client-streamer pair
  - [ ] Multiple client-streamer pairs
  - [ ] STUN packet timeout handling
- [ ] Performance testing
  - [ ] Concurrent connection count
  - [ ] Data forwarding latency

---

## Open Questions

1. ~~Port prediction vs query trade-off?~~ → Solved by Method C
2. ~~Multi-instance coordination needed?~~ → Single port needs no coordination
3. ~~Port range size recommendation?~~ → Single port needs no port range
4. **STUN packet buffering memory management strategy?**
5. **Pairing timeout cleanup mechanism?**
6. **Need to support ICE Lite mode (unidirectional STUN)?**