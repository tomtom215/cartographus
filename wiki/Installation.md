# Installation Guide

Detailed installation instructions for all supported deployment methods.

**[Home](Home)** | **[Quick Start](Quick-Start)** | **Installation** | **[Configuration](Configuration)**

---

## Deployment Options

| Method | Best For | Complexity |
|--------|----------|------------|
| **[Docker Compose](#docker-compose)** | Most users, homelabs | Easy |
| **[Docker Run](#docker-run)** | Quick testing | Easy |
| **[Unraid](#unraid)** | Unraid users | Easy |
| **[Kubernetes](#kubernetes)** | Enterprise, HA setups | Advanced |
| **[Binary](#binary-installation)** | Bare metal, custom setups | Advanced |

---

## System Requirements

### Minimum Requirements

| Component | Requirement |
|-----------|-------------|
| **Architecture** | x86_64 (amd64) or ARM64 |
| **RAM** | 1 GB available |
| **Disk** | 500 MB for application + database growth |
| **OS** | Linux, macOS, or Windows with Docker |

### Recommended Specifications

| Deployment Size | RAM | CPU | Storage |
|-----------------|-----|-----|---------|
| Small (1 server, < 10 users) | 1 GB | 1 core | 1 GB |
| Medium (2-3 servers, < 50 users) | 2 GB | 2 cores | 5 GB |
| Large (5+ servers, 100+ users) | 4 GB | 4 cores | 20 GB |

### Architecture Support

| Architecture | Support | Notes |
|--------------|---------|-------|
| x86_64 (amd64) | Full | Recommended |
| ARM64 (aarch64) | Full | Raspberry Pi 4+, Apple Silicon |
| ARMv7 | Not Supported | DuckDB limitation |

---

## Docker Compose

The recommended installation method for most users.

### Prerequisites

- Docker Engine 20.10+
- Docker Compose V2

### Basic Installation

1. **Create a directory:**

   ```bash
   mkdir -p ~/cartographus && cd ~/cartographus
   ```

2. **Create docker-compose.yml:**

   ```yaml
   services:
     cartographus:
       image: ghcr.io/tomtom215/cartographus:latest
       container_name: cartographus
       ports:
         - "3857:3857"
       environment:
         # Required: Security
         - JWT_SECRET=${JWT_SECRET}
         - ADMIN_USERNAME=${ADMIN_USERNAME:-admin}
         - ADMIN_PASSWORD=${ADMIN_PASSWORD}

         # Required: At least one media server
         - ENABLE_PLEX_SYNC=${ENABLE_PLEX_SYNC:-false}
         - PLEX_URL=${PLEX_URL:-}
         - PLEX_TOKEN=${PLEX_TOKEN:-}

         # Optional: Logging
         - LOG_LEVEL=${LOG_LEVEL:-info}
       volumes:
         - ./data:/data
       restart: unless-stopped
       healthcheck:
         test: ["CMD", "curl", "-f", "http://localhost:3857/api/v1/health"]
         interval: 30s
         timeout: 10s
         retries: 3
         start_period: 30s
   ```

3. **Create .env file:**

   ```bash
   # Generate JWT secret
   JWT_SECRET=$(openssl rand -base64 48)

   cat > .env << EOF
   JWT_SECRET=${JWT_SECRET}
   ADMIN_USERNAME=admin
   ADMIN_PASSWORD=YourSecurePassword123!

   # Plex configuration
   ENABLE_PLEX_SYNC=true
   PLEX_URL=http://plex:32400
   PLEX_TOKEN=your_plex_token
   EOF
   ```

4. **Start the container:**

   ```bash
   docker-compose up -d
   ```

5. **Verify installation:**

   ```bash
   # Check logs
   docker logs cartographus

   # Check health
   curl http://localhost:3857/api/v1/health
   ```

### Updating

```bash
docker-compose pull
docker-compose up -d
```

---

## Docker Run

For quick testing or simple deployments without Compose.

```bash
docker run -d \
  --name cartographus \
  -p 3857:3857 \
  -e JWT_SECRET="$(openssl rand -base64 48)" \
  -e ADMIN_USERNAME=admin \
  -e ADMIN_PASSWORD=YourSecurePassword123! \
  -e ENABLE_PLEX_SYNC=true \
  -e PLEX_URL=http://your-plex:32400 \
  -e PLEX_TOKEN=your_plex_token \
  -v cartographus_data:/data \
  --restart unless-stopped \
  ghcr.io/tomtom215/cartographus:latest
```

---

## Unraid

Cartographus can be installed via Docker on Unraid.

### Manual Docker Installation

1. Go to **Docker** tab in Unraid
2. Click **Add Container**
3. Configure:
   - **Name**: `cartographus`
   - **Repository**: `ghcr.io/tomtom215/cartographus:latest`
   - **Network Type**: `bridge`
   - **Port Mapping**: `3857` -> `3857`
   - **Path Mapping**: `/data` -> `/mnt/user/appdata/cartographus`
4. Add environment variables (see [Configuration](Configuration))
5. Click **Apply**

### Template

A community template may be available in the Unraid Community Applications store. Search for "Cartographus" or "Map".

See [deploy/unraid/README.md](https://github.com/tomtom215/cartographus/blob/main/deploy/unraid/README.md) for detailed Unraid instructions.

---

## Kubernetes

Experimental Kubernetes support is available for high-availability deployments.

### Prerequisites

- Kubernetes 1.25+
- kubectl configured
- Persistent volume provisioner (for data storage)

### Basic Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cartographus
  labels:
    app: cartographus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cartographus
  template:
    metadata:
      labels:
        app: cartographus
    spec:
      containers:
        - name: cartographus
          image: ghcr.io/tomtom215/cartographus:latest
          ports:
            - containerPort: 3857
          env:
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: cartographus-secrets
                  key: jwt-secret
            - name: ADMIN_USERNAME
              value: "admin"
            - name: ADMIN_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: cartographus-secrets
                  key: admin-password
          volumeMounts:
            - name: data
              mountPath: /data
          livenessProbe:
            httpGet:
              path: /api/v1/health
              port: 3857
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /api/v1/health
              port: 3857
            initialDelaySeconds: 5
            periodSeconds: 5
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: cartographus-data
---
apiVersion: v1
kind: Service
metadata:
  name: cartographus
spec:
  selector:
    app: cartographus
  ports:
    - port: 3857
      targetPort: 3857
  type: ClusterIP
```

See [deploy/kubernetes/README.md](https://github.com/tomtom215/cartographus/blob/main/deploy/kubernetes/README.md) for complete manifests.

---

## Binary Installation

For bare-metal installations without Docker.

### Prerequisites

- Go 1.24+ (for building)
- GCC (CGO is required for DuckDB)
- Node.js 20+ (for frontend build)

### Build from Source

```bash
# Clone repository
git clone https://github.com/tomtom215/cartographus.git
cd map

# Set required environment variables
export GOTOOLCHAIN=local
export CGO_ENABLED=1

# Build backend
go build -tags "wal,nats" -o cartographus ./cmd/server

# Build frontend
cd web && npm ci && npm run build && cd ..

# Run
./cartographus
```

### Environment Setup

Create a configuration file or set environment variables:

```bash
export JWT_SECRET="your-secure-secret-at-least-32-chars"
export ADMIN_USERNAME="admin"
export ADMIN_PASSWORD="YourSecurePassword123!"
export ENABLE_PLEX_SYNC="true"
export PLEX_URL="http://localhost:32400"
export PLEX_TOKEN="your-plex-token"
export DUCKDB_PATH="/var/lib/cartographus/cartographus.duckdb"

./cartographus
```

### Systemd Service

```ini
[Unit]
Description=Cartographus Media Analytics
After=network.target

[Service]
Type=simple
User=cartographus
Group=cartographus
WorkingDirectory=/opt/cartographus
ExecStart=/opt/cartographus/cartographus
Restart=on-failure
RestartSec=5
EnvironmentFile=/etc/cartographus/cartographus.env

[Install]
WantedBy=multi-user.target
```

---

## Data Persistence

### Volume Structure

The `/data` directory contains all persistent data:

```
/data/
├── cartographus.duckdb          # Main analytics database
├── cartographus.duckdb.wal      # Database write-ahead log
├── wal/                # BadgerDB event durability
├── nats/               # NATS JetStream storage
├── backups/            # Automated backups
└── sessions/           # Session store (if file-based)
```

### Backup Recommendations

- Mount `/data` to a persistent volume
- Enable automated backups (see [Configuration](Configuration))
- Back up the entire `/data` directory regularly

---

## Verification

After installation, verify Cartographus is running correctly:

### Health Check

```bash
curl http://localhost:3857/api/v1/health
```

Expected response:

```json
{
  "status": "healthy",
  "version": "...",
  "database": "connected",
  "uptime": "..."
}
```

### Logs

```bash
# Docker
docker logs cartographus

# Binary
journalctl -u cartographus -f
```

### Web Interface

Open http://localhost:3857 in your browser and log in with your admin credentials.

---

## Next Steps

- **[Configuration](Configuration)** - Complete configuration reference
- **[Media Servers](Media-Servers)** - Connect your media servers
- **[Reverse Proxy](Reverse-Proxy)** - Set up HTTPS access
- **[Troubleshooting](Troubleshooting)** - Common issues and solutions
