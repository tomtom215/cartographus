# Cartographus Wiki

Welcome to the Cartographus documentation. This wiki provides comprehensive guides for installing, configuring, and using Cartographus - a data analytics and geographic visualization platform for self-hosted media servers.

---

## What is Cartographus?

Cartographus transforms your media server data into actionable insights with interactive maps, 47+ charts, and real-time monitoring. It connects directly to Plex, Jellyfin, and Emby servers to visualize playback activity geographically.

**Key highlights:**
- **302 REST API endpoints** for comprehensive analytics
- **Geographic visualization** with WebGL maps and 3D globe
- **Real-time monitoring** via WebSocket connections
- **Security detection** for account sharing and suspicious activity
- **Self-hosted** - your data stays on your server

---

## Quick Navigation

### Getting Started

| Guide | Description | Time |
|-------|-------------|------|
| **[Quick Start](Quick-Start)** | Get running with Docker in 5 minutes | 5 min |
| **[Installation](Installation)** | Detailed installation options and requirements | 15 min |
| **[First Steps](First-Steps)** | What to do after installation | 10 min |

### Configuration

| Guide | Description |
|-------|-------------|
| **[Configuration Reference](Configuration)** | Complete environment variable reference |
| **[Media Servers](Media-Servers)** | Connect Plex, Jellyfin, Emby, or Tautulli |
| **[Authentication](Authentication)** | JWT, OIDC, Plex OAuth, and multi-auth setup |
| **[Reverse Proxy](Reverse-Proxy)** | Nginx, Caddy, and Traefik examples |

### Using Cartographus

| Guide | Description |
|-------|-------------|
| **[Features](Features)** | Complete feature overview with screenshots |
| **[Analytics](Analytics)** | Understanding the analytics dashboards |
| **[Maps & Globe](Maps-and-Globe)** | Geographic visualization guide |
| **[Security Detection](Security-Detection)** | Account sharing and anomaly detection |

### Reference

| Guide | Description |
|-------|-------------|
| **[API Reference](API-Reference)** | REST API documentation (302 endpoints) |
| **[Troubleshooting](Troubleshooting)** | Common issues and solutions |
| **[FAQ](FAQ)** | Frequently asked questions |

---

## Who Is This For?

Cartographus is designed for:

| User Type | Use Case |
|-----------|----------|
| **Self-hosters** | Full control over your media analytics data |
| **Homelabbers** | Add geographic visualization to your homelab stack |
| **Server admins** | Monitor multiple media servers from one dashboard |
| **Tinkerers** | Extensive API for custom integrations |
| **First-timers** | Docker-based setup with sensible defaults |

---

## System Requirements

### Minimum

| Component | Requirement |
|-----------|-------------|
| **Architecture** | x86_64 (amd64) or ARM64 |
| **RAM** | 1 GB |
| **Storage** | 500 MB + data |
| **Docker** | 20.10+ (recommended) |

### Recommended

| Component | Requirement |
|-----------|-------------|
| **RAM** | 2-4 GB |
| **Storage** | SSD with 10+ GB free |
| **Network** | Stable connection to media servers |

> **Note**: ARM v7 is not supported due to DuckDB limitations.

---

## Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Backend | Go 1.24 | High-performance single binary |
| Database | DuckDB 1.4.3 | Analytics with spatial extensions |
| Frontend | TypeScript 5.9 | Type-safe UI |
| Maps | MapLibre GL JS | WebGL vector rendering |
| 3D Globe | deck.gl | Large-scale visualization |
| Charts | ECharts 6.0 | Interactive dashboards |

---

## Getting Help

- **[Troubleshooting Guide](Troubleshooting)** - Common issues and solutions
- **[FAQ](FAQ)** - Frequently asked questions
- **[GitHub Issues](https://github.com/tomtom215/cartographus/issues)** - Bug reports and feature requests
- **[GitHub Discussions](https://github.com/tomtom215/cartographus/discussions)** - Community support

---

## License

Cartographus is open source software licensed under the [AGPL-3.0 License](https://github.com/tomtom215/cartographus/blob/main/LICENSE).
