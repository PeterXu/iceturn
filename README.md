# IceTurn

Custom TURN server with single-port multiplexing and ICE-ufrag pairing.

## Overview

IceTurn is a modified TURN server based on pion/turn that supports:
- ✅ Single port for unlimited client-streamer pairs
- ✅ Automatic pairing via ICE-ufrag parsing
- ✅ Standard ICE Full mode compatibility
- ✅ No browser modifications required
- ✅ STUN packet interception and buffering
- ✅ Docker deployment ready

## Status

✅ **Core Implementation Complete & Tested**

### Implemented Components

- [x] Configuration management
- [x] STUN Binding Request parser
- [x] ice-ufrag to allocation mapping
- [x] STUN packet buffering manager
- [x] Pairing logic
- [x] STUN packet interception (wrapper)
- [x] Server integration
- [x] Unit tests
- [x] Docker support

### Pending

- [ ] Performance testing with multiple concurrent pairs
- [ ] Production monitoring
- [ ] TLS support

## Quick Start

### Native Build

```bash
# Build
make build

# Run
./bin/icecoordinator

# Test
make test
```

### Docker Deployment (Recommended)

```bash
# Using Docker Compose
docker-compose up -d

# View logs
docker logs -f iceturn

# Stop
docker-compose down
```

See [DOCKER.md](DOCKER.md) for detailed Docker deployment guide.

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

## Configuration

### Environment Variables

```bash
# Required
PUBLIC_IP=192.168.1.100

# Optional
REALM=iceturn.local
STUN_PENDING_TIMEOUT=30s

# Credentials
TURN_CREDENTIALS=iceturn:iceturn
```

### Configuration File

See [config.yaml](config.yaml) for example.

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
├── Dockerfile                 # Docker image
├── docker-compose.yml         # Docker Compose config
└── config.yaml                # Example configuration
```

## Deployment

- **Docker**: See [DOCKER.md](DOCKER.md)
- **Integration Test**: See [INTEGRATION_TEST.md](INTEGRATION_TEST.md)

## Tested With

- ✅ Chrome (latest)
- ✅ Firefox (latest)
- ✅ Safari (latest)
