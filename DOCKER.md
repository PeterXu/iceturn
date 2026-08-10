# Docker Deployment Guide

This guide explains how to deploy IceTurn using Docker.

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Build and start
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

### Using Docker directly

```bash
# Build image
docker build -t iceturn:latest .

# Run container
docker run -d \
  --name iceturn \
  -p 3478:3478/udp \
  -p 3478:3478/tcp \
  -e PUBLIC_IP=your-public-ip \
  -e TURN_USER_magpie=magpie \
  iceturn:latest

# View logs
docker logs -f iceturn

# Stop
docker stop iceturn
docker rm iceturn
```

## Configuration

### Environment Variables

IceTurn can be configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `REALM` | TURN realm | `magpie.local` |
| `PUBLIC_IP` | Public IP for relay address | `127.0.0.1` |
| `LISTEN_UDP` | UDP listen address | `:3478` |
| `LISTEN_TCP` | TCP listen address | `:3478` |
| `STUN_PENDING_TIMEOUT` | STUN packet buffer timeout | `30s` |
| `TURN_CREDENTIALS` | Credentials (e.g., `user1:pass1,user2:pass2`) | - |

### Example with Environment Variables

```bash
docker run -d \
  --name iceturn \
  -p 3478:3478/udp \
  -p 3478:3478/tcp \
  -e REALM=mycompany.local \
  -e PUBLIC_IP=192.168.1.100 \
  -e TURN_CREDENTIALS=magpie:secret123,client1:pass456 \
  iceturn:latest
```

### Using config.yaml

You can also mount a configuration file:

```bash
docker run -d \
  --name iceturn \
  -p 3478:3478/udp \
  -p 3478:3478/tcp \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  iceturn:latest
```

**Note:** Environment variables override config file values.

## Integration with Magpie

### Using Docker Compose with Magpie

Create a combined `docker-compose.yml`:

```yaml
version: '3.8'

services:
  iceturn:
    build: ./iceturn
    ports:
      - "3478:3478/udp"
      - "3478:3478/tcp"
    environment:
      - PUBLIC_IP=${MAGPIE_HOST_IP}
      - TURN_USER_magpie=magpie
    networks:
      - magpie-network

  magpie:
    build: ./magpie
    ports:
      - "8080:8080"
    environment:
      - MAGPIE_HOST_IP=${MAGPIE_HOST_IP}
      - MAGPIE_TURN_USER=magpie
      - MAGPIE_TURN_PASS=magpie
      - MAGPIE_NO_SANDBOX=1
    depends_on:
      - iceturn
    networks:
      - magpie-network

networks:
  magpie-network:
    driver: bridge
```

Create `.env`:

```bash
MAGPIE_HOST_IP=192.168.1.100  # Your LAN IP
```

Run:

```bash
docker-compose up -d
```

### Manual Integration

1. **Start IceTurn:**

```bash
docker run -d \
  --name iceturn \
  -p 3478:3478/udp \
  -p 3478:3478/tcp \
  -e PUBLIC_IP=192.168.1.100 \
  -e TURN_CREDENTIALS=magpie:magpie \
  iceturn:latest
```

2. **Start Magpie:**

```bash
docker run -d \
  --name magpie \
  -p 8080:8080 \
  -e MAGPIE_HOST_IP=192.168.1.100 \
  -e MAGPIE_TURN_USER=magpie \
  -e MAGPIE_TURN_PASS=magpie \
  -e MAGPIE_NO_SANDBOX=1 \
  magpie:latest
```

## Production Deployment

### Resource Limits

```yaml
services:
  iceturn:
    # ...
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 256M
        reservations:
          cpus: '0.5'
          memory: 128M
```

### Health Check

The Dockerfile includes a health check:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD netstat -tuln | grep 3478 || exit 1
```

Check health status:

```bash
docker ps
# Look for "healthy" status
```

### Logging

View logs:

```bash
# Real-time logs
docker logs -f iceturn

# Last 100 lines
docker logs --tail 100 iceturn

# Logs with timestamps
docker logs -t iceturn
```

Persist logs:

```yaml
volumes:
  - ./logs:/var/log/iceturn
```

### TLS/SSL (Future)

IceTurn currently doesn't support TLS. For production use:
- Deploy behind a TLS-enabled reverse proxy
- Or wait for TLS support in future versions

## Troubleshooting

### Container won't start

```bash
# Check logs
docker logs iceturn

# Check if port is already in use
lsof -i :3478

# Try without port mapping
docker run -d --name iceturn-test iceturn:latest
```

### Can't connect from browser

1. **Check PUBLIC_IP:**
   ```bash
   docker exec iceturn env | grep PUBLIC_IP
   ```

2. **Check firewall:**
   ```bash
   # Allow port 3478
   sudo ufw allow 3478/udp
   sudo ufw allow 3478/tcp
   ```

3. **Check network:**
   ```bash
   # From another machine
   telnet your-server-ip 3478
   ```

### Performance issues

```bash
# Check container stats
docker stats iceturn

# Increase resources
docker-compose down
# Edit docker-compose.yml to add resource limits
docker-compose up -d
```

## Advanced Configuration

### Multiple Credentials

Using environment variables:

```bash
docker run -d \
  -e TURN_CREDENTIALS=magpie:password1,client1:password2,streamer1:password3 \
  iceturn:latest
```

Using config.yaml:

```yaml
credentials:
  magpie: password1
  client1: password2
  streamer1: password3
```

### Custom Realm

```bash
docker run -d \
  -e REALM=mycompany.com \
  iceturn:latest
```

### Debug Mode

```bash
# Run interactively
docker run -it --rm \
  -e REALM=debug.local \
  iceturn:latest

# Shell into container
docker exec -it iceturn sh
```

## Building

### Multi-architecture Build

```bash
# Build for multiple platforms
docker buildx build --platform linux/amd64,linux/arm64 -t iceturn:latest .
```

### Versioned Build

```bash
docker build -t iceturn:v1.0.0 .
docker tag iceturn:v1.0.0 iceturn:latest
```

## Monitoring (Future)

Planned features:
- Prometheus metrics endpoint
- Grafana dashboard
- Health check API

## Backup and Restore

No persistent data to backup. Configuration is provided via:
- Environment variables
- Mounted config.yaml

To "backup", save your configuration:

```bash
# Export environment variables
docker inspect iceturn | jq '.[0].Config.Env'

# Or save config.yaml
docker cp iceturn:/app/config.yaml ./config.yaml.backup
```

## Security Considerations

1. **Use strong passwords** for TURN credentials
2. **Restrict access** using firewall rules
3. **Monitor logs** for suspicious activity
4. **Update regularly** to get security fixes
5. **Use HTTPS** in production (reverse proxy)

## Next Steps

- Test with real WebRTC traffic
- Monitor performance
- Set up log aggregation
- Configure alerts