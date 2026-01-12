# Quick Start Guide

Get Cartographus running in under 5 minutes with Docker.

**[Home](Home)** | **Quick Start** | **[Installation](Installation)** | **[Configuration](Configuration)**

---

## Prerequisites

- Docker and Docker Compose installed
- At least one media server (Plex, Jellyfin, or Emby)
- Network access between Cartographus and your media server

---

## Step 1: Create Directory

```bash
mkdir cartographus && cd cartographus
```

---

## Step 2: Create docker-compose.yml

Choose your media server and copy the appropriate configuration:

### For Plex Users

```yaml
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    container_name: cartographus
    ports:
      - "3857:3857"
    environment:
      # Security (required)
      - JWT_SECRET=replace_with_random_string_at_least_32_characters_long
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD=YourSecurePassword123!
      # Plex connection
      - ENABLE_PLEX_SYNC=true
      - PLEX_URL=http://your-plex-server:32400
      - PLEX_TOKEN=your_plex_token_here
      - ENABLE_PLEX_REALTIME=true
    volumes:
      - ./data:/data
    restart: unless-stopped
```

### For Jellyfin Users

```yaml
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    container_name: cartographus
    ports:
      - "3857:3857"
    environment:
      # Security (required)
      - JWT_SECRET=replace_with_random_string_at_least_32_characters_long
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD=YourSecurePassword123!
      # Jellyfin connection
      - JELLYFIN_ENABLED=true
      - JELLYFIN_URL=http://your-jellyfin-server:8096
      - JELLYFIN_API_KEY=your_jellyfin_api_key
      - JELLYFIN_REALTIME_ENABLED=true
    volumes:
      - ./data:/data
    restart: unless-stopped
```

### For Emby Users

```yaml
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    container_name: cartographus
    ports:
      - "3857:3857"
    environment:
      # Security (required)
      - JWT_SECRET=replace_with_random_string_at_least_32_characters_long
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD=YourSecurePassword123!
      # Emby connection
      - EMBY_ENABLED=true
      - EMBY_URL=http://your-emby-server:8096
      - EMBY_API_KEY=your_emby_api_key
      - EMBY_REALTIME_ENABLED=true
    volumes:
      - ./data:/data
    restart: unless-stopped
```

---

## Step 3: Generate a Secure JWT Secret

Replace the placeholder JWT_SECRET with a secure random string:

```bash
# Generate a 48-character random secret
openssl rand -base64 48
```

Copy the output and replace `replace_with_random_string_at_least_32_characters_long` in your docker-compose.yml.

---

## Step 4: Get Your Media Server Credentials

### Plex Token

1. Sign in to Plex Web at https://app.plex.tv
2. Open any media item and click "Get Info"
3. Click "View XML"
4. Find `X-Plex-Token=` in the URL

Or use this command if you have `curl` and `jq`:

```bash
curl -s "https://plex.tv/api/v2/users/signin" \
  -X POST \
  -H "X-Plex-Client-Identifier: cartographus" \
  -d "login=YOUR_PLEX_USERNAME&password=YOUR_PLEX_PASSWORD" | \
  jq -r '.authToken'
```

### Jellyfin API Key

1. Open Jellyfin Dashboard
2. Go to **Administration** > **API Keys**
3. Click **+** to create a new key
4. Name it "Cartographus" and copy the key

### Emby API Key

1. Open Emby Dashboard
2. Go to **Settings** > **API Keys**
3. Click **New API Key**
4. Name it "Cartographus" and copy the key

---

## Step 5: Start Cartographus

```bash
docker-compose up -d
```

---

## Step 6: Access the Interface

Open your browser and navigate to:

```
http://localhost:3857
```

Log in with:
- **Username**: `admin` (or what you set in ADMIN_USERNAME)
- **Password**: `YourSecurePassword123!` (or what you set in ADMIN_PASSWORD)

---

## What's Next?

Cartographus will begin syncing data from your media server. Depending on your library size, this may take a few minutes.

### Recommended Next Steps

1. **[First Steps](First-Steps)** - Explore the interface and understand the dashboards
2. **[Media Servers](Media-Servers)** - Configure additional servers or advanced options
3. **[Authentication](Authentication)** - Set up OIDC or Plex Sign-In for multi-user access
4. **[Reverse Proxy](Reverse-Proxy)** - Expose Cartographus securely with HTTPS

---

## Troubleshooting Quick Start Issues

| Issue | Solution |
|-------|----------|
| "Connection refused" to media server | Verify the URL and that both containers can reach each other. Use container names for Docker networking. |
| "Invalid token" or "Unauthorized" | Double-check your Plex token or API key. Regenerate if needed. |
| "JWT secret too short" | Generate a new secret with at least 32 characters using `openssl rand -base64 48`. |
| Page loads but no data | Wait 2-3 minutes for initial sync. Check logs: `docker logs cartographus`. |
| Maps not displaying | Ensure your browser supports WebGL. Try Chrome or Firefox. |

For more detailed troubleshooting, see the **[Troubleshooting Guide](Troubleshooting)**.

---

## Example: Multiple Media Servers

You can connect Plex, Jellyfin, and Emby simultaneously:

```yaml
services:
  cartographus:
    image: ghcr.io/tomtom215/cartographus:latest
    container_name: cartographus
    ports:
      - "3857:3857"
    environment:
      - JWT_SECRET=your_secure_jwt_secret_at_least_32_chars
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD=YourSecurePassword123!
      # Plex
      - ENABLE_PLEX_SYNC=true
      - PLEX_URL=http://plex:32400
      - PLEX_TOKEN=your_plex_token
      - ENABLE_PLEX_REALTIME=true
      # Jellyfin
      - JELLYFIN_ENABLED=true
      - JELLYFIN_URL=http://jellyfin:8096
      - JELLYFIN_API_KEY=your_jellyfin_key
      - JELLYFIN_REALTIME_ENABLED=true
      # Emby
      - EMBY_ENABLED=true
      - EMBY_URL=http://emby:8096
      - EMBY_API_KEY=your_emby_key
      - EMBY_REALTIME_ENABLED=true
    volumes:
      - ./data:/data
    restart: unless-stopped
```

Cartographus automatically deduplicates users who appear on multiple servers.

---

**Next:** [Installation](Installation) for detailed installation options, or [Configuration](Configuration) for all available settings.
