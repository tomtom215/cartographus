# Reverse Proxy Configuration

Expose Cartographus securely with HTTPS using popular reverse proxies.

**[Home](Home)** | **[Configuration](Configuration)** | **[Authentication](Authentication)** | **Reverse Proxy**

---

## Overview

Running Cartographus behind a reverse proxy provides:

- **HTTPS encryption** - Secure connections
- **Domain name** - Access via `cartographus.example.com`
- **WebSocket support** - Real-time updates
- **Authentication integration** - SSO passthrough

---

## Important: WebSocket Support

Cartographus uses WebSocket connections for real-time updates. Your reverse proxy **must** support WebSocket upgrade.

Key headers for WebSocket:

```
Upgrade: websocket
Connection: upgrade
```

---

## Nginx

### Basic Configuration

```nginx
server {
    listen 443 ssl http2;
    server_name cartographus.example.com;

    ssl_certificate /etc/letsencrypt/live/cartographus.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/cartographus.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:3857;
        proxy_http_version 1.1;

        # WebSocket support
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Forward headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeout for WebSocket
        proxy_read_timeout 86400;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name cartographus.example.com;
    return 301 https://$server_name$request_uri;
}
```

### With Docker Compose

```yaml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
      - ./certs:/etc/letsencrypt:ro
    depends_on:
      - cartographus

  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    # No ports exposed - accessed via nginx
    environment:
      - TRUSTED_PROXIES=172.16.0.0/12
      # ... other config
```

---

## Caddy

Caddy automatically provisions HTTPS certificates.

### Basic Configuration

```
cartographus.example.com {
    reverse_proxy localhost:3857
}
```

That's it! Caddy handles:
- HTTPS certificate provisioning
- Certificate renewal
- WebSocket upgrade
- HTTP to HTTPS redirect

### With Docker Compose

```yaml
services:
  caddy:
    image: caddy:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
    depends_on:
      - cartographus

  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    environment:
      - TRUSTED_PROXIES=172.16.0.0/12

volumes:
  caddy_data:
```

---

## Traefik

### Docker Labels

```yaml
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.cartographus.rule=Host(`cartographus.example.com`)"
      - "traefik.http.routers.cartographus.entrypoints=websecure"
      - "traefik.http.routers.cartographus.tls.certresolver=letsencrypt"
      - "traefik.http.services.cartographus.loadbalancer.server.port=3857"
    environment:
      - TRUSTED_PROXIES=172.16.0.0/12
    networks:
      - traefik

networks:
  traefik:
    external: true
```

### Static Configuration

```yaml
# traefik.yml
entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
  websecure:
    address: ":443"

certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@example.com
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
```

---

## NPM (Nginx Proxy Manager)

1. Add a new Proxy Host
2. **Domain Names**: `cartographus.example.com`
3. **Scheme**: `http`
4. **Forward Hostname/IP**: `cartographus` (container name) or IP
5. **Forward Port**: `3857`
6. **Websockets Support**: Enable
7. **SSL**: Request a new certificate

---

## Cloudflare Tunnel

For exposing without opening ports.

### cloudflared Configuration

```yaml
tunnel: your-tunnel-id
credentials-file: /etc/cloudflared/credentials.json

ingress:
  - hostname: cartographus.example.com
    service: http://localhost:3857
  - service: http_status:404
```

### Docker Compose

```yaml
services:
  cloudflared:
    image: cloudflare/cloudflared:latest
    command: tunnel run
    environment:
      - TUNNEL_TOKEN=your-tunnel-token
    depends_on:
      - cartographus

  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    # No ports exposed
```

---

## Trusted Proxies Configuration

When behind a reverse proxy, configure trusted proxies so Cartographus correctly identifies client IPs:

```yaml
environment:
  # Single proxy
  - TRUSTED_PROXIES=172.17.0.1

  # Docker network range
  - TRUSTED_PROXIES=172.16.0.0/12

  # Multiple proxies
  - TRUSTED_PROXIES=172.17.0.1,10.0.0.0/8,192.168.0.0/16
```

This enables proper:
- Rate limiting per real client IP
- Geolocation for real client IP
- Logging of real client IP

---

## CORS Configuration

If accessing from a different domain:

```yaml
environment:
  # Specific origins
  - CORS_ORIGINS=https://dashboard.example.com,https://admin.example.com

  # Or allow all (not recommended for production)
  - CORS_ORIGINS=*
```

---

## Common Issues

### WebSocket Not Working

**Symptoms**: Real-time updates don't work, map doesn't update live.

**Solutions**:

1. Ensure WebSocket upgrade headers are set:
   ```nginx
   proxy_set_header Upgrade $http_upgrade;
   proxy_set_header Connection "upgrade";
   ```

2. Increase timeout for long-lived connections:
   ```nginx
   proxy_read_timeout 86400;
   ```

3. Disable buffering:
   ```nginx
   proxy_buffering off;
   ```

### Wrong Client IP in Logs

**Symptoms**: All requests show proxy IP instead of real client IP.

**Solution**: Configure `TRUSTED_PROXIES` with your proxy's IP or network range.

### Mixed Content Warnings

**Symptoms**: HTTPS page tries to load HTTP resources.

**Solution**: Ensure `X-Forwarded-Proto` header is set:
```nginx
proxy_set_header X-Forwarded-Proto $scheme;
```

### 502 Bad Gateway

**Symptoms**: Nginx returns 502 errors.

**Solutions**:

1. Verify Cartographus is running:
   ```bash
   docker ps | grep cartographus
   ```

2. Check if the port is correct (3857)

3. If using container names, ensure they're on the same Docker network

---

## Security Headers

Add security headers in your reverse proxy:

### Nginx

```nginx
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-XSS-Protection "1; mode=block" always;
add_header Referrer-Policy "strict-origin-when-cross-origin" always;
```

### Caddy

```
cartographus.example.com {
    header {
        X-Frame-Options "SAMEORIGIN"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Referrer-Policy "strict-origin-when-cross-origin"
    }
    reverse_proxy localhost:3857
}
```

---

## Next Steps

- **[Authentication](Authentication)** - Configure user authentication
- **[Configuration](Configuration)** - All configuration options
- **[Troubleshooting](Troubleshooting)** - Common issues
