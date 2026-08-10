# CLAUDE.md

## What is iceturn

TURN server with single-port multiplexing based on pion/turn. Supports unlimited clients on a single relay port.

## Key Insight

**pion/turn provides everything needed:**
- Single-port multiplexing via SO_REUSEPORT
- Allocation management via FiveTuple
- Data forwarding via XOR-PEER-ADDRESS in Data Indication
- Complete TURN protocol implementation (RFC 5766)

**No custom logic needed!**

## Commands

```bash
# Build
make build

# Run
./bin/icecoordinator

# Run with custom config
PUBLIC_IP=192.168.1.100 ./bin/icecoordinator

# Test
make test

# Docker
docker-compose up -d
```

## Architecture — Simple and Standard

```
WebRTC Client/Streamer
  │
  └─ TURN Allocate (standard TURN protocol)
  └─ Send/ChannelData (standard TURN protocol)
      │
      ↓
pion/turn Server
  ├─ FiveTuple-based allocation tracking
  ├─ Permission/Channel management
  └─ Data forwarding with XOR-PEER-ADDRESS
```

## How Clients Are Distinguished

**Key: Data Indication contains sender's address**

```
Client1 → Send Indication:
  - XOR-PEER-ADDRESS: Streamer's address
  - DATA: actual data

TURN → Data Indication:
  - XOR-PEER-ADDRESS: Client1's address ✅
  - DATA: actual data

Streamer receives:
  - Knows it's from Client1 ✅
```

Even with same relay port, TURN protocol distinguishes clients via XOR-PEER-ADDRESS.

## Module Map

| Path | Role |
|---|---|
| `cmd/icecoordinator/main.go` | Entry point. Simple wrapper around pion/turn. |
| `internal/config/config.go` | Configuration management. |

## Configuration

```bash
# Required
PUBLIC_IP=192.168.1.100

# Optional
REALM=iceturn.local
LISTEN_UDP=:3478
LISTEN_TCP=:3478

# Credentials
TURN_CREDENTIALS=user1:pass1,user2:pass2
```

## Implementation Status

✅ **Complete and Simple**

- [x] TURN server with pion/turn
- [x] Configuration management
- [x] Single-port multiplexing (SO_REUSEPORT)
- [x] Standard TURN protocol

## Why It's Simple

**Original misunderstanding:**
- Thought needed custom STUN interception
- Thought needed ice-ufrag pairing
- Thought needed custom forwarding logic

**Reality:**
- pion/turn uses SO_REUSEPORT for single-port multiplexing
- FiveTuple uniquely identifies allocations
- Data Indication contains XOR-PEER-ADDRESS for sender identification
- Standard TURN protocol handles everything

## References

- **RFC 5766** - TURN Protocol
- **RFC 5389** - STUN Protocol
- **pion/turn** - https://github.com/pion/turn
- **Implementation Notes** - docs/TURN_IMPLEMENTATION.md

## Design Documentation

- **Original Design** - docs/2026-08-10-custom-turn-server-design.md (for historical context)
- **Actual Implementation** - docs/TURN_IMPLEMENTATION.md (how it really works)