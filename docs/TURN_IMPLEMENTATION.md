# TURN Server Implementation

> Date: 2026-08-10
> Status: ✅ Simplified

## Overview

iceturn is a simplified TURN server using pion/turn with single-port multiplexing support.

## Key Insight

**pion/turn already provides everything needed:**

1. **Single-port multiplexing** via SO_REUSEPORT
2. **Allocation management** via FiveTuple (SrcAddr, DstAddr, Protocol)
3. **Data forwarding** via TURN protocol (XOR-PEER-ADDRESS in Data Indication)
4. **Complete TURN protocol** implementation (RFC 5766)

## How It Works

### Client1 → TURN → Streamer Flow

```
Client1 sends Send Indication:
  ├─ XOR-PEER-ADDRESS: Streamer's address
  ├─ DATA: actual data
  └─ Source: Client1_IP:Client1_Port

TURN Server:
  ├─ Finds allocation via FiveTuple
  ├─ Creates Data Indication:
  │   ├─ XOR-PEER-ADDRESS: Client1_IP:Client1_Port ✅
  │   └─ DATA: actual data
  └─ Sends to Streamer

Streamer receives:
  └─ XOR-PEER-ADDRESS tells who sent the data ✅
```

### Differentiation Mechanism

**Even with same relay port:**
- Data Indication contains XOR-PEER-ADDRESS
- Streamer knows which client sent each packet
- No need for ice-ufrag pairing or custom routing

## What Was Removed

### Unnecessary Complexity

1. **STUN interception** - pion/turn handles TURN protocol
2. **ice-ufrag pairing** - TURN protocol already routes correctly
3. **Custom forwarding** - pion/turn's standard forwarding works
4. **Allocation management** - pion/turn manages allocations

### Why It Was Wrong

Initial design misunderstood:
- Thought needed custom routing for single-port
- Didn't realize TURN protocol handles this via XOR-PEER-ADDRESS
- pion/turn uses SO_REUSEPORT for multiple allocations on same port

## Current Implementation

### Simplicity

```go
// main.go - all you need
server, _ := turn.NewServer(turn.ServerConfig{
    Realm: "iceturn.local",
    AuthHandler: authHandler,
    PacketConnConfigs: []turn.PacketConnConfig{{
        PacketConn: udpListener,
        RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
            RelayAddress: net.ParseIP(publicIP),
            Address:      "0.0.0.0",
        },
    }},
    // ... TCP config
})
```

### What pion/turn Provides

1. **Allocation Management**
   - FiveTuple-based allocation tracking
   - Permission and channel binding
   - Lifetime management

2. **Data Forwarding**
   - Send Indication → Data Indication
   - ChannelData forwarding
   - Proper XOR-PEER-ADDRESS handling

3. **Single-Port Multiplexing**
   - SO_REUSEPORT support
   - Multiple allocations on same port
   - Correct routing via FiveTuple

## References

### RFC 5766 - TURN Protocol

Key concept:
> Data Indication MUST contain both XOR-PEER-ADDRESS and DATA.
> XOR-PEER-ADDRESS is the source transport address.

This is how clients distinguish senders!

### pion/turn Implementation

Key code in allocation.go:
```go
// Build Data Indication with sender's address
peerAddressAttr := proto.PeerAddress{IP: udpAddr.IP, Port: udpAddr.Port}
dataAttr := proto.Data(buffer[:n])

msg, _ := stun.Build(
    stun.TransactionID,
    stun.NewType(stun.MethodData, stun.ClassIndication),
    peerAddressAttr,  // XOR-PEER-ADDRESS
    dataAttr,          // DATA
)
```

## Conclusion

**Simplicity wins:**
- ~50 lines of code vs ~1000+ lines
- Standard TURN protocol
- No custom logic needed
- pion/turn handles everything

**Lesson learned:**
- Always check what libraries already provide
- Understand protocols before implementing custom logic
- Test assumptions with real implementations