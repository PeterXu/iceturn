# IceTurn

Custom TURN server with single-port multiplexing and ICE-ufrag pairing.

## Overview

IceTurn is a modified TURN server based on pion/turn that supports:
- ✅ Single port for unlimited client-streamer pairs
- ✅ Automatic pairing via ICE-ufrag parsing
- ✅ Standard ICE Full mode compatibility
- ✅ No browser modifications required
- ✅ STUN packet interception and buffering

## Status

✅ **Core Implementation Complete**

### Implemented Components

- [x] Configuration management
- [x] STUN Binding Request parser
- [x] ice-ufrag to allocation mapping
- [x] STUN packet buffering manager
- [x] Pairing logic
- [x] STUN packet interception (wrapper)
- [x] Server integration
- [x] Unit tests

### Pending

- [ ] Integration testing with real WebRTC
- [ ] Allocation tracking in pion/turn callbacks
- [ ] Data forwarding based on pairing

## Quick Start

```bash
# Build
make build

# Run
./bin/icecoordinator

# Test
make test
```

## Architecture

```
WebRTC Client/Streamer (browser)
  │
  ├─ TURN Allocate (standard credential)
  │
  └─ STUN Binding Request (ice-ufrag)
      ↓
STUNInterceptConn (intercepts packets)
      ↓
IceTurnServer.handleSTUNPacket()
  ├─ Parse ice-ufrag: "remote:local"
  ├─ Register mapping
  ├─ Buffer first STUN packet
  └─ Establish pair when both sides sent STUN
      ↓
pion/turn Server (normal TURN processing)
```

## Design

See [Design Document](docs/2026-08-10-custom-turn-server-design.md)

## Project Structure

```
iceturn/
├── cmd/icecoordinator/        # Main program
├── internal/
│   ├── server/                # IceTurn server wrapper
│   ├── stunparse/             # STUN packet parsing + interception
│   ├── allocation/            # ice-ufrag mapping manager
│   ├── pending/               # STUN packet buffering
│   └── config/                # Configuration
├── docs/                      # Design documents
└── config.yaml                # Example configuration
```