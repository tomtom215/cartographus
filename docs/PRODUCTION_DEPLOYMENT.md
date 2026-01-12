# Production Deployment Guide

This guide covers deploying Cartographus in production environments with best practices for reliability, performance, and security.

**Related Documentation**:
- [CLAUDE.md](../CLAUDE.md) - Project overview and development guide
- [SECURITY_HARDENING.md](./SECURITY_HARDENING.md) - Security configuration guide
- [MONITORING.md](./MONITORING.md) - Monitoring and alerting
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) - Common issues and solutions

---

## Table of Contents

1. [Pre-Deployment Checklist](#pre-deployment-checklist)
2. [Resource Requirements](#resource-requirements)
3. [Deployment Options](#deployment-options)
4. [Configuration for Production](#configuration-for-production)
5. [High Availability](#high-availability)
6. [Load Balancing](#load-balancing)
7. [Scaling Guidelines](#scaling-guidelines)
8. [Backup and Disaster Recovery](#backup-and-disaster-recovery)
9. [Maintenance Procedures](#maintenance-procedures)
10. [Rollback Procedures](#rollback-procedures)

---

## Pre-Deployment Checklist

Complete these checks before deploying to production:

### Security Configuration

- [ ] **JWT_SECRET** is a random 32+ character string (generate with `openssl rand -base64 32`)
- [ ] **ADMIN_PASSWORD** is strong (16+ chars, mixed case, numbers, symbols)
- [ ] **AUTH_MODE** is set to `jwt` (never `none` in production)
- [ ] **CORS_ORIGINS** lists only your specific domains (never `*`)
- [ ] All media server API keys are valid and have minimal required permissions
- [ ] TLS/HTTPS is configured (see [Load Balancing](#load-balancing))

### Infrastructure

- [ ] Persistent storage is provisioned for `/data` volume
- [ ] Backup destination is configured and accessible
- [ ] Monitoring endpoints are accessible by monitoring systems
- [ ] Firewall rules allow traffic on port 3857 (or configured port)
- [ ] Reverse proxy/load balancer is configured (if applicable)

### Media Server Connectivity

- [ ] All configured media servers are reachable from deployment network
- [ ] WebSocket connections can be established (no firewall blocking upgrades)
- [ ] API keys tested with `curl` or similar tool

### Monitoring and Alerting

- [ ] Health check endpoint verified: `GET /api/v1/health`
- [ ] Prometheus metrics endpoint accessible: `GET /metrics`
- [ ] Log aggregation configured (structured JSON logs)
- [ ] Alerting rules configured for critical metrics

---

## Resource Requirements

### Minimum Requirements

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 2 cores | 4+ cores |
| RAM | 2 GB | 4-8 GB |
| Disk | 20 GB SSD | 100+ GB SSD |
| Network | 10 Mbps | 100+ Mbps |

### Sizing by Scale

#### Small Deployment (1-5 users, 1-2 media servers)

```yaml
resources:
  requests:
    cpu: "500m"
    memory: "1Gi"
  limits:
    cpu: "2"
    memory: "2Gi"
```

**Environment:**
```bash
DUCKDB_MAX_MEMORY=1GB
SYNC_BATCH_SIZE=500
API_MAX_PAGE_SIZE=50
```

#### Medium Deployment (5-50 users, 2-5 media servers)

```yaml
resources:
  requests:
    cpu: "1"
    memory: "2Gi"
  limits:
    cpu: "4"
    memory: "4Gi"
```

**Environment:**
```bash
DUCKDB_MAX_MEMORY=2GB
SYNC_BATCH_SIZE=1000
API_MAX_PAGE_SIZE=100
```

#### Large Deployment (50+ users, 5+ media servers)

```yaml
resources:
  requests:
    cpu: "2"
    memory: "4Gi"
  limits:
    cpu: "8"
    memory: "8Gi"
```

**Environment:**
```bash
DUCKDB_MAX_MEMORY=4GB
SYNC_BATCH_SIZE=2000
API_MAX_PAGE_SIZE=200
```

### Disk Space Estimation

| Data Type | Size per 1000 events | Retention |
|-----------|---------------------|-----------|
| Playback events | ~5 MB | Configurable |
| Geolocation cache | ~10 MB | 7 days |
| WAL entries | ~2 MB | Until confirmed |
| Backups | ~20 MB (compressed) | Based on policy |
| JetStream storage | ~50 MB | 7 days |

**Formula:** Estimated disk = (daily_events / 1000) * 5 MB * retention_days + 500 MB base

---

## Deployment Options

### Docker Compose (Recommended for Single Server)

```yaml
# docker-compose.yml
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:v1.0.0  # Pin specific version
    container_name: cartographus
    restart: unless-stopped
    ports:
      - "127.0.0.1:3857:3857"  # Bind to localhost, use reverse proxy
    environment:
      # Security (REQUIRED)
      - JWT_SECRET=${JWT_SECRET}
      - ADMIN_USERNAME=${ADMIN_USERNAME}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD}
      - AUTH_MODE=jwt
      - CORS_ORIGINS=https://youromain.com

      # Performance
      - DUCKDB_MAX_MEMORY=2GB
      - LOG_LEVEL=info
      - LOG_FORMAT=json

      # Media Servers
      - PLEX_URL=${PLEX_URL}
      - PLEX_TOKEN=${PLEX_TOKEN}
      - ENABLE_PLEX_SYNC=true
      - ENABLE_PLEX_REALTIME=true
    volumes:
      - cartographus_data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3857/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    logging:
      driver: json-file
      options:
        max-size: "100m"
        max-file: "5"
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G

volumes:
  cartographus_data:
    driver: local
```

### Kubernetes (Recommended for Multi-Server)

See [deploy/kubernetes/README.md](../deploy/kubernetes/README.md) for complete Kubernetes manifests.

**Key considerations:**
- Use Deployments with replicas: 1 (single writer for DuckDB)
- Configure PersistentVolumeClaim for data storage
- Use Ingress with TLS termination
- Set resource requests and limits

### Docker Swarm

Docker Swarm provides native clustering for Docker containers. Since DuckDB is single-writer, use `replicas: 1` with placement constraints.

```yaml
# docker-compose.swarm.yml
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:v1.0.0
    deploy:
      replicas: 1
      placement:
        constraints:
          - node.role == manager  # Or use a specific node label
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
        window: 120s
      resources:
        limits:
          cpus: '4'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
      update_config:
        parallelism: 1
        delay: 10s
        failure_action: rollback
        order: stop-first
      rollback_config:
        parallelism: 1
        delay: 10s
    environment:
      - JWT_SECRET_FILE=/run/secrets/jwt_secret
      - ADMIN_PASSWORD_FILE=/run/secrets/admin_password
      - AUTH_MODE=jwt
      - CORS_ORIGINS=https://yourdomain.com
      - DUCKDB_MAX_MEMORY=2GB
      - LOG_LEVEL=info
      - LOG_FORMAT=json
    volumes:
      - cartographus_data:/data
    networks:
      - proxy
      - internal
    secrets:
      - jwt_secret
      - admin_password
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3857/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s

networks:
  proxy:
    external: true
  internal:
    driver: overlay
    internal: true

volumes:
  cartographus_data:
    driver: local
    driver_opts:
      type: none
      o: bind
      device: /srv/cartographus/data

secrets:
  jwt_secret:
    external: true
  admin_password:
    external: true
```

**Deploy to Swarm:**

```bash
# 1. Create secrets
echo "your-jwt-secret-min-32-chars" | docker secret create jwt_secret -
echo "your-secure-admin-password" | docker secret create admin_password -

# 2. Create overlay network (if not exists)
docker network create --driver overlay proxy

# 3. Deploy the stack
docker stack deploy -c docker-compose.swarm.yml cartographus

# 4. Verify deployment
docker service ls
docker service logs cartographus_cartographus

# 5. Scale (if needed for load balancer, but only 1 writer)
# Note: Only scale to 1 due to DuckDB single-writer limitation
docker service scale cartographus_cartographus=1
```

**Swarm with Traefik:**

```yaml
# Add labels for Traefik in Swarm mode
services:
  cartographus:
    # ... (same as above)
    deploy:
      labels:
        - "traefik.enable=true"
        - "traefik.http.routers.cartographus.rule=Host(`cartographus.example.com`)"
        - "traefik.http.routers.cartographus.entrypoints=websecure"
        - "traefik.http.routers.cartographus.tls.certresolver=letsencrypt"
        - "traefik.http.services.cartographus.loadbalancer.server.port=3857"
        - "traefik.docker.network=proxy"
```

**Important Swarm Considerations:**
- Use Docker secrets instead of environment variables for sensitive data
- Mount data volume on a specific node or use NFS/GlusterFS for shared storage
- The `stop-first` update order ensures clean shutdown before starting new container
- Place on manager node or labeled node for predictable storage location

### Binary Deployment (Bare Metal)

```bash
# 1. Download latest release
curl -LO https://github.com/tomtom215/cartographus/releases/latest/download/cartographus-linux-amd64

# 2. Make executable
chmod +x cartographus-linux-amd64

# 3. Create data directory
sudo mkdir -p /var/lib/cartographus
sudo chown $USER:$USER /var/lib/cartographus

# 4. Create systemd service
sudo tee /etc/systemd/system/cartographus.service << EOF
[Unit]
Description=Cartographus Media Analytics
After=network.target

[Service]
Type=simple
User=cartographus
Group=cartographus
WorkingDirectory=/opt/cartographus
ExecStart=/opt/cartographus/cartographus-linux-amd64
Restart=always
RestartSec=10

# Environment file
EnvironmentFile=/etc/cartographus/env

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/cartographus

[Install]
WantedBy=multi-user.target
EOF

# 5. Enable and start
sudo systemctl daemon-reload
sudo systemctl enable cartographus
sudo systemctl start cartographus
```

---

## Configuration for Production

### Critical Environment Variables

```bash
# ===========================================
# SECURITY (Required)
# ===========================================
AUTH_MODE=jwt
JWT_SECRET=<generated-32+-char-secret>
ADMIN_USERNAME=admin
ADMIN_PASSWORD=<strong-password>
CORS_ORIGINS=https://yourdomain.com

# ===========================================
# PERFORMANCE
# ===========================================
DUCKDB_MAX_MEMORY=2GB
SYNC_BATCH_SIZE=1000
SYNC_INTERVAL=5m
API_DEFAULT_PAGE_SIZE=20
API_MAX_PAGE_SIZE=100

# ===========================================
# LOGGING (Production)
# ===========================================
LOG_LEVEL=info
LOG_FORMAT=json

# ===========================================
# RELIABILITY
# ===========================================
SYNC_RETRY_ATTEMPTS=5
SYNC_RETRY_DELAY=2s
NATS_MAX_RECONNECT=5
NATS_CIRCUIT_BREAKER_THRESHOLD=5

# ===========================================
# BACKUP (Required for Production)
# ===========================================
BACKUP_ENABLED=true
BACKUP_SCHEDULE_ENABLED=true
BACKUP_INTERVAL=24h
BACKUP_RETENTION_MIN_COUNT=3
BACKUP_RETENTION_MAX_DAYS=90
BACKUP_COMPRESSION_ENABLED=true
BACKUP_NOTIFY_FAILURE=true

# ===========================================
# WAL (Recommended for Durability)
# ===========================================
WAL_ENABLED=true
WAL_SYNC_WRITES=true
WAL_RETRY_INTERVAL=30s
WAL_MAX_RETRIES=100
```

### Configuration File (config.yaml)

For complex deployments, use a YAML configuration file:

```yaml
# /etc/cartographus/config.yaml
server:
  host: "0.0.0.0"
  port: 3857
  timeout: 30s

database:
  path: /data/cartographus.duckdb
  max_memory: 2GB

auth:
  mode: jwt
  session_timeout: 24h

logging:
  level: info
  format: json

# Multi-server configuration
jellyfin_servers:
  - enabled: true
    server_id: jellyfin-primary
    url: http://jellyfin1.internal:8096
    api_key: ${JELLYFIN_API_KEY_1}
    realtime_enabled: true

  - enabled: true
    server_id: jellyfin-secondary
    url: http://jellyfin2.internal:8096
    api_key: ${JELLYFIN_API_KEY_2}
    realtime_enabled: true

backup:
  enabled: true
  schedule_enabled: true
  interval: 24h
  preferred_hour: 2
  retention:
    min_count: 3
    max_count: 50
    max_days: 90
```

---

## High Availability

### Important Limitation

Cartographus uses DuckDB as its primary database, which is an **embedded** database designed for single-writer access. This means:

- Only one Cartographus instance can write to a DuckDB database at a time
- Read replicas are not supported
- Horizontal scaling of write operations is not possible

### High Availability Strategies

#### Strategy 1: Active-Passive (Recommended)

Run two instances with shared storage, but only one active at a time:

```yaml
# Primary instance (active)
cartographus-primary:
  image: ghcr.io/tomtom215/cartographus:v1.0.0
  volumes:
    - /mnt/shared/cartographus:/data
  deploy:
    mode: replicated
    replicas: 1

# Secondary instance (standby - manually activated on failure)
cartographus-secondary:
  image: ghcr.io/tomtom215/cartographus:v1.0.0
  volumes:
    - /mnt/shared/cartographus:/data
  deploy:
    mode: replicated
    replicas: 0  # Scale up when primary fails
```

**Failover procedure:**
1. Stop primary instance
2. Ensure storage is properly unmounted/synced
3. Start secondary instance
4. Update DNS/load balancer to point to secondary

#### Strategy 2: Automated Failover with Keepalived

```bash
# /etc/keepalived/keepalived.conf (Primary)
vrrp_script check_cartographus {
    script "/usr/bin/curl -sf http://localhost:3857/api/v1/health"
    interval 5
    weight -2
    fall 3
    rise 2
}

vrrp_instance VI_CARTOGRAPHUS {
    state MASTER
    interface eth0
    virtual_router_id 51
    priority 100
    advert_int 1

    authentication {
        auth_type PASS
        auth_pass secret
    }

    virtual_ipaddress {
        192.168.1.100/24
    }

    track_script {
        check_cartographus
    }

    notify_master "/usr/local/bin/cartographus-start.sh"
    notify_backup "/usr/local/bin/cartographus-stop.sh"
}
```

### Data Durability

Enable WAL for maximum data durability:

```bash
WAL_ENABLED=true
WAL_SYNC_WRITES=true
WAL_ENTRY_TTL=168h  # 7 days
```

With WAL enabled, events are persisted to BadgerDB before being processed, surviving:
- Process crashes
- NATS failures
- Power outages (with sync writes)

---

## Load Balancing

### Reverse Proxy Configuration

Always deploy Cartographus behind a reverse proxy for:
- TLS termination
- Rate limiting
- Request logging
- WebSocket handling

#### Nginx Configuration

```nginx
upstream cartographus {
    server 127.0.0.1:3857;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name cartographus.example.com;

    # TLS Configuration
    ssl_certificate /etc/letsencrypt/live/cartographus.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cartographus.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-Frame-Options DENY always;
    add_header X-XSS-Protection "1; mode=block" always;

    # WebSocket support
    location /api/v1/ws {
        proxy_pass http://cartographus;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;
    }

    # Regular HTTP
    location / {
        proxy_pass http://cartographus;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";

        # Timeouts
        proxy_connect_timeout 10s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name cartographus.example.com;
    return 301 https://$server_name$request_uri;
}
```

#### Traefik Configuration

```yaml
# docker-compose.yml with Traefik
services:
  traefik:
    image: traefik:v3.0
    command:
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.email=admin@example.com"
      - "--certificatesresolvers.letsencrypt.acme.storage=/letsencrypt/acme.json"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - letsencrypt:/letsencrypt

  cartographus:
    image: ghcr.io/tomtom215/cartographus:v1.0.0
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.cartographus.rule=Host(`cartographus.example.com`)"
      - "traefik.http.routers.cartographus.entrypoints=websecure"
      - "traefik.http.routers.cartographus.tls.certresolver=letsencrypt"
      - "traefik.http.services.cartographus.loadbalancer.server.port=3857"
```

#### HAProxy Configuration

HAProxy provides high-performance load balancing with advanced health checking. This configuration supports WebSocket connections and proper client IP forwarding.

```haproxy
# /etc/haproxy/haproxy.cfg

global
    log /dev/log local0
    log /dev/log local1 notice
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin
    stats timeout 30s
    user haproxy
    group haproxy
    daemon

    # TLS settings
    ssl-default-bind-ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384
    ssl-default-bind-ciphersuites TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256
    ssl-default-bind-options ssl-min-ver TLSv1.2 no-tls-tickets

defaults
    log     global
    mode    http
    option  httplog
    option  dontlognull
    option  forwardfor
    option  http-server-close
    timeout connect 5000
    timeout client  50000
    timeout server  50000
    timeout tunnel  3600000  # 1 hour for WebSocket
    errorfile 400 /etc/haproxy/errors/400.http
    errorfile 403 /etc/haproxy/errors/403.http
    errorfile 408 /etc/haproxy/errors/408.http
    errorfile 500 /etc/haproxy/errors/500.http
    errorfile 502 /etc/haproxy/errors/502.http
    errorfile 503 /etc/haproxy/errors/503.http
    errorfile 504 /etc/haproxy/errors/504.http

# Stats page (optional, for monitoring)
listen stats
    bind *:8404
    stats enable
    stats uri /stats
    stats refresh 10s
    stats admin if LOCALHOST

# Frontend for HTTPS
frontend https_front
    bind *:443 ssl crt /etc/haproxy/certs/cartographus.pem alpn h2,http/1.1

    # Redirect HTTP to HTTPS
    http-request redirect scheme https unless { ssl_fc }

    # Security headers
    http-response set-header Strict-Transport-Security "max-age=31536000; includeSubDomains"
    http-response set-header X-Content-Type-Options nosniff
    http-response set-header X-Frame-Options DENY
    http-response set-header X-XSS-Protection "1; mode=block"

    # ACL for WebSocket
    acl is_websocket hdr(Upgrade) -i websocket
    acl is_websocket_path path_beg /api/v1/ws

    # Use WebSocket backend for WebSocket requests
    use_backend cartographus_ws if is_websocket or is_websocket_path

    # Default backend
    default_backend cartographus_http

# Frontend for HTTP (redirect to HTTPS)
frontend http_front
    bind *:80
    http-request redirect scheme https code 301

# Backend for regular HTTP requests
backend cartographus_http
    balance roundrobin
    option httpchk GET /api/v1/health
    http-check expect status 200

    # Forward client IP
    http-request set-header X-Real-IP %[src]
    http-request set-header X-Forwarded-For %[src]
    http-request set-header X-Forwarded-Proto https

    server cartographus1 127.0.0.1:3857 check inter 5s fall 3 rise 2

# Backend for WebSocket connections
backend cartographus_ws
    balance source  # Sticky sessions for WebSocket
    option httpchk GET /api/v1/health
    http-check expect status 200

    # Forward client IP
    http-request set-header X-Real-IP %[src]
    http-request set-header X-Forwarded-For %[src]
    http-request set-header X-Forwarded-Proto https

    # WebSocket-specific settings
    timeout tunnel 3600000  # 1 hour timeout for WebSocket

    server cartographus1 127.0.0.1:3857 check inter 5s fall 3 rise 2
```

**HAProxy TLS Certificate Setup:**

```bash
# Combine certificate and private key for HAProxy
cat /etc/letsencrypt/live/cartographus.example.com/fullchain.pem \
    /etc/letsencrypt/live/cartographus.example.com/privkey.pem \
    > /etc/haproxy/certs/cartographus.pem

# Set proper permissions
chmod 600 /etc/haproxy/certs/cartographus.pem

# Test configuration
haproxy -c -f /etc/haproxy/haproxy.cfg

# Reload HAProxy
systemctl reload haproxy
```

**HAProxy with Docker Compose:**

```yaml
# docker-compose.yml with HAProxy
services:
  haproxy:
    image: haproxy:2.9-alpine
    ports:
      - "80:80"
      - "443:443"
      - "8404:8404"  # Stats page
    volumes:
      - ./haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro
      - ./certs:/etc/haproxy/certs:ro
    depends_on:
      - cartographus
    restart: unless-stopped

  cartographus:
    image: ghcr.io/tomtom215/cartographus:v1.0.0
    environment:
      - TRUSTED_PROXIES=172.17.0.0/16
    # ... other configuration
```

### Trusted Proxies

Configure trusted proxy IPs to ensure correct client IP detection:

```bash
# For Docker networks
TRUSTED_PROXIES=172.17.0.0/16,10.0.0.0/8

# For Cloudflare (add to existing)
TRUSTED_PROXIES=172.17.0.0/16,173.245.48.0/20,103.21.244.0/22,103.22.200.0/22,103.31.4.0/22
```

---

## Scaling Guidelines

### Vertical Scaling

DuckDB performs best with more RAM for query caching:

| Metric | Action |
|--------|--------|
| High query latency | Increase `DUCKDB_MAX_MEMORY` |
| Slow sync operations | Increase CPU cores |
| Disk I/O bottleneck | Switch to NVMe SSD |

### Connection Limits

Adjust based on concurrent users:

```bash
# For high concurrency
HTTP_TIMEOUT=60s
RATE_LIMIT_REQUESTS=1000
RATE_LIMIT_WINDOW=1m
```

### Monitoring Thresholds

Set alerts for these metrics:

| Metric | Warning | Critical |
|--------|---------|----------|
| CPU Usage | >70% | >90% |
| Memory Usage | >80% | >95% |
| Disk Usage | >70% | >90% |
| Request Latency (p95) | >500ms | >2s |
| Error Rate | >1% | >5% |
| WebSocket Connections | >80% capacity | >95% capacity |

---

## Backup and Disaster Recovery

### Backup Configuration

```bash
# Production backup settings
BACKUP_ENABLED=true
BACKUP_SCHEDULE_ENABLED=true
BACKUP_INTERVAL=24h
BACKUP_PREFERRED_HOUR=2          # 2 AM
BACKUP_TYPE=full

# Retention policy
BACKUP_RETENTION_MIN_COUNT=3     # Always keep at least 3
BACKUP_RETENTION_MAX_DAYS=90     # Delete backups older than 90 days
BACKUP_RETENTION_KEEP_DAILY_DAYS=7
BACKUP_RETENTION_KEEP_WEEKLY_WEEKS=4
BACKUP_RETENTION_KEEP_MONTHLY_MONTHS=6

# Compression and encryption
BACKUP_COMPRESSION_ENABLED=true
BACKUP_COMPRESSION_LEVEL=6
BACKUP_ENCRYPTION_ENABLED=true
BACKUP_ENCRYPTION_KEY=<store-securely>

# Notifications
BACKUP_NOTIFY_FAILURE=true
BACKUP_WEBHOOK_URL=https://hooks.slack.com/services/xxx
```

### Backup Verification

Regularly verify backups can be restored:

```bash
# 1. List available backups
curl -s http://localhost:3857/api/v1/backup/list | jq

# 2. Test restore (to temporary location)
docker run --rm -it \
  -v /path/to/backup:/backup:ro \
  -v /tmp/test-restore:/data \
  ghcr.io/tomtom215/cartographus:v1.0.0 \
  /cartographus --restore /backup/backup-2024-01-01.tar.gz

# 3. Verify restored data
docker run --rm -it \
  -v /tmp/test-restore:/data \
  ghcr.io/tomtom215/cartographus:v1.0.0 \
  /cartographus --verify
```

### Disaster Recovery Procedure

**RTO (Recovery Time Objective):** 15-30 minutes
**RPO (Recovery Point Objective):** Last backup (typically 24 hours)

1. **Assess the situation**
   - Identify root cause
   - Determine scope of data loss

2. **Provision new infrastructure**
   - Deploy fresh Cartographus instance
   - Configure with same environment variables

3. **Restore from backup**
   ```bash
   # Copy backup to new instance
   scp backup-latest.tar.gz new-server:/tmp/

   # Restore
   docker exec cartographus \
     curl -X POST http://localhost:3857/api/v1/backup/restore \
     -F "backup=@/tmp/backup-latest.tar.gz"
   ```

4. **Verify restoration**
   - Check health endpoint
   - Verify data integrity
   - Test media server connectivity

5. **Update DNS/routing**
   - Point traffic to new instance

6. **Post-mortem**
   - Document incident
   - Identify improvements

---

## Maintenance Procedures

### Planned Maintenance Window

1. **Notify users** (if applicable)
2. **Create pre-maintenance backup**
   ```bash
   curl -X POST http://localhost:3857/api/v1/backup/create?type=full
   ```
3. **Stop sync operations**
   - Set `SYNC_INTERVAL=0` temporarily
4. **Perform maintenance**
5. **Restart service**
6. **Verify health**
   ```bash
   curl http://localhost:3857/api/v1/health
   ```
7. **Re-enable sync**

### Version Upgrades

1. **Review changelog** for breaking changes
2. **Backup current data**
3. **Pull new image** (for Docker)
   ```bash
   docker pull ghcr.io/tomtom215/cartographus:v1.1.0
   ```
4. **Test in staging** (if available)
5. **Update production**
   ```bash
   docker-compose up -d
   ```
6. **Verify upgrade**
   ```bash
   curl http://localhost:3857/api/v1/health
   ```

### Database Maintenance

DuckDB handles most maintenance automatically. For manual operations:

```bash
# Compact database (reduces size after deletions)
curl -X POST http://localhost:3857/api/v1/admin/compact

# Analyze tables (update statistics for query optimizer)
# This happens automatically, but can be triggered manually
curl -X POST http://localhost:3857/api/v1/admin/analyze
```

---

## Rollback Procedures

### Quick Rollback (Docker)

```bash
# 1. Stop current version
docker-compose down

# 2. Update image tag to previous version
# Edit docker-compose.yml: image: ghcr.io/tomtom215/cartographus:v1.0.0

# 3. Start previous version
docker-compose up -d

# 4. Verify
curl http://localhost:3857/api/v1/health
```

### Data Rollback

If data corruption occurred:

1. **Stop the application**
2. **Restore from backup**
   ```bash
   curl -X POST http://localhost:3857/api/v1/backup/restore \
     -F "backup=@/path/to/pre-upgrade-backup.tar.gz"
   ```
3. **Restart with previous version**
4. **Verify data integrity**

### Rollback Checklist

- [ ] Application starts successfully
- [ ] Health endpoint returns healthy
- [ ] WebSocket connections working
- [ ] Media server sync functioning
- [ ] Historical data accessible
- [ ] No error spikes in logs

---

## Support

For issues not covered in this guide:

1. Check [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)
2. Review application logs: `docker logs cartographus`
3. Open an issue: https://github.com/tomtom215/cartographus/issues
