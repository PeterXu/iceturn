# CLAUDE.md

## What is iceturn

Custom TURN server with single-port multiplexing and ICE-ufrag pairing. Based on pion/turn, supports unlimited client-streamer pairs on a single port.

**Key innovation:** Parse STUN Binding Request to extract ice-ufrag, automatically pair allocations without needing different relay addresses.

## Commands

```bash
# Build
make build

# Run with default config
./bin/icecoordinator

# Run with custom config
./bin/icecoordinator --config config.yaml

# Test
make test

# Clean
make clean

# Download dependencies
make deps
```

## Architecture — Single-port multiplexing with ice-ufrag pairing

```
WebRTC Client/Streamer (browser)
  │
  ├─ TURN Allocate (standard credential)
  │
  └─ STUN Binding Request (ice-ufrag)
      ↓
IceTurn Server (modified pion/turn)
  │
  ├─ Parse STUN Binding Request
  │   └─ Extract ice-ufrag: "remote-ufrag:local-ufrag"
  │
  ├─ Register ice-ufrag → Allocation mapping
  │
  ├─ Buffer first STUN packet (waiting for pair)
  │
  ├─ When both sides sent STUN:
  │   └─ Establish pair: Allocation-A ↔ Allocation-B
  │
  └─ Forward data based on pairing relationship
```

**Core principle:**
- ICE Full mode: both client and streamer send STUN Binding Request
- STUN Binding Request contains: `USERNAME: "remote-ufrag:local-ufrag"`
- TURN Server parses to establish pairing
- No need for different relay addresses

## Module map

| Path | Role |
|---|---|
| `cmd/icecoordinator/main.go` | Entry point. Creates IceTurnServer and starts server. |
| `internal/server/server.go` | IceTurn server wrapper. Integrates STUN interception with pion/turn. Handles STUN Binding Request. |
| `internal/stunparse/intercept.go` | STUN packet interception wrapper. Intercepts ReadFrom calls to process STUN messages. |
| `internal/stunparse/stunparse.go` | STUN Binding Request parser. Extracts ice-ufrag from USERNAME attribute. |
| `internal/allocation/manager.go` | Manages ice-ufrag → Allocation mappings and pairing state. |
| `internal/pending/pending.go` | Buffers STUN packets until pairing is established. Includes timeout cleanup. |
| `internal/config/config.go` | Configuration management. Default config and validation. |

## Key types

```go
// STUN parsing result
type ParseResult struct {
    RemoteUfrag string  // remote ice-ufrag
    LocalUfrag  string  // local ice-ufrag
    Message     *stun.Message
}

// Allocation mapping manager
type Manager struct {
    iceUfragMap  map[string]*Allocation  // ice-ufrag → Allocation
    pairMap      map[string]*PairState   // pairing state
    allocToUfrag map[*Allocation]string  // reverse mapping
}

// Pending STUN packet manager
type Manager struct {
    pending map[string]*PendingSTUN  // local-ufrag → buffered packet
    timeout time.Duration
}
```

## Implementation status

### ✅ Implemented

- [x] Configuration management
- [x] STUN Binding Request parser
- [x] ice-ufrag → Allocation mapping manager
- [x] STUN packet buffering manager
- [x] Pairing logic
- [x] Cleanup routines
- [x] **STUN packet interception wrapper**
- [x] **IceTurn server integration**
- [x] **Connection tracking**
- [x] **Unit tests**

### ⚠️ Pending integration

- [ ] **Test with real WebRTC browsers**
  - Need to verify STUN interception works correctly
  - Test pairing establishment with real browsers
  - Verify data forwarding

- [ ] **Improve allocation tracking**
  - Hook into pion/turn allocation events
  - Register allocation when created
  - Remove allocation when closed

- [ ] **Data forwarding optimization**
  - Verify forwarding based on pairing works
  - Handle edge cases (allocation close, timeout, etc.)

### 🧪 Testing needed

- [x] Unit tests for each component
- [ ] Integration test: single client-streamer pair
- [ ] Integration test: multiple client-streamer pairs
- [ ] Compatibility test with real WebRTC browsers

## Design flow (from design doc)

```
Streamer                         TURN Server                          Client
   │                                 │                                    │
   │ 1. Allocate Request            │                                    │
   │    USERNAME: "turn-user-s"     │                                    │
   ├────────────────────────────────>│ Record: turn-user-s → Allocation-A │
   │                                 │                                    │
   │ 2. Allocate Success            │                                    │
   │    RELAYED: 192.168.1.1:3478   │                                    │
   │<────────────────────────────────┤                                    │
   │                                 │                                    │
   │                                 │         3. Allocate Request        │
   │                                 │<───────────────────────────────────┤
   │                                 │         Record: turn-user-c → Allocation-B
   │                                 │                                    │
   │                                 │         4. Allocate Success        │
   │                                 ├───────────────────────────────────>│
   │                                 │                                    │
   │                                 │         5. STUN Binding Request    │
   │                                 │            USERNAME: "s-ufrag:c-ufrag"
   │                                 │<───────────────────────────────────┤
   │                                 │                                    │
   │                                 │ Parse: client-ufrag → Allocation-B │
   │                                 │ Buffer STUN packet                 │
   │                                 │                                    │
   │ 6. STUN Binding Request        │                                    │
   │    USERNAME: "c-ufrag:s-ufrag" │                                    │
   ├────────────────────────────────>│                                    │
   │                                 │ Parse: streamer-ufrag → Allocation-A
   │                                 │                                    │
   │                                 │ Establish pair: A ↔ B              │
   │                                 │ Forward buffered STUN packet       │
   │                                 │                                    │
   │ 7-10. STUN Binding exchanges   │                                    │
   │                                 │ (forward based on pairing)         │
   │                                 │                                    │
   │ 11. DTLS/SRTP data             │                                    │
   ├────────────────────────────────>│ Forward to paired allocation       │
```

## Configuration

```yaml
# config.yaml
realm: iceturn.local

# Single port for all connections
listen_udp: ":3478"
listen_tcp: ":3478"

# Public IP for relay address
public_ip: "192.168.1.100"

# STUN packet buffering timeout
stun_pending_timeout: 30s

# Authentication credentials
credentials:
  streamer-1: "password1"
  client-1: "password2"
```

## Key implementation: STUN packet interception

**Solution: Wrap PacketConn**

We implemented a `STUNInterceptConn` wrapper (in `internal/stunparse/intercept.go`) that:

1. Wraps the underlying `net.PacketConn`
2. Intercepts `ReadFrom()` calls
3. Checks if incoming data is a STUN message
4. Calls `stunHandler` callback for STUN packets
5. Processes or forwards the packet

**Integration flow:**

```
UDP/TCP Listener
      ↓
STUNInterceptConn (wrapper)
      ↓ (intercepts ReadFrom)
IsSTUNMessage?
  Yes → handleSTUNPacket()
         ├─ ParseBindingRequest()
         ├─ Register ice-ufrag
         ├─ TryEstablishPair()
         └─ Buffer if needed
  No  → Forward to pion/turn
      ↓
pion/turn Server (normal processing)
```

**Code location:**
- Wrapper: `internal/stunparse/intercept.go`
- Handler: `internal/server/server.go:handleSTUNPacket()`
- Integration: `internal/server/server.go:createTURNServer()`

**Benefits:**
- ✅ No modification to pion/turn internals
- ✅ Clean separation of concerns
- ✅ Easy to test and debug
- ✅ Can disable interception if needed

## Dependencies

- `github.com/pion/turn/v5` - TURN server implementation
- `github.com/pion/stun/v3` - STUN message parsing
- `github.com/pion/logging` - Logging framework

No external dependencies beyond pion ecosystem.

## Design documentation

- **Design doc**: `docs/2026-08-10-custom-turn-server-design.md`
- **Method C**: Single-port multiplexing + ICE-ufrag pairing

## Browser compatibility

**ICE Full mode support:**
- ✅ Chrome (default)
- ✅ Firefox (default)
- ✅ Safari (default)
- ✅ Edge (default)

All major browsers use ICE Full mode by default, so both client and streamer will send STUN Binding Request.

## Next steps

1. **Integrate STUN interception** (critical)
   - Decide on approach (wrap PacketConn vs modify pion/turn)
   - Implement packet interception
   - Call `handleSTUNPacket()` for STUN messages

2. **Implement allocation lookup**
   - Track connection → Allocation mapping
   - Required for registering ice-ufrag mapping

3. **Add tests**
   - Unit tests for core components
   - Integration test with real WebRTC

4. **Performance testing**
   - Concurrent connections
   - Memory usage (pending STUN buffers)

## Code conventions

- Go 1.21+
- Standard Go project layout (`cmd/`, `internal/`)
- Use `sync.RWMutex` for thread-safe maps
- Return `bool` for existence checks (e.g., `GetAllocation() (*Allocation, bool)`)
- Use `time.Duration` for timeouts
- Start cleanup routines with `StartCleanupRoutine()` pattern
- Log with `github.com/pion/logging`

## Similar projects

- **coturn**: Standard TURN server (multi-port, no ice-ufrag pairing)
- **pion/turn**: Base TURN implementation (this project modifies it)
- **eturnal**: Erlang TURN server

## References

- RFC 5766: TURN protocol
- RFC 5389: STUN protocol
- ICE Full mode: RFC 8445
- pion/turn: https://github.com/pion/turn