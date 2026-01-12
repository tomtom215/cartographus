# Kubernetes Deployment

**Last Verified**: 2026-01-11

> **EXPERIMENTAL**: These manifests are provided as a starting point and have not been
> tested in a production Kubernetes environment. Please review and adapt them to your
> specific cluster configuration. Community feedback and contributions are welcome.

This directory contains Kubernetes manifests for deploying Cartographus.

## Prerequisites

- Kubernetes cluster (1.21+)
- kubectl configured
- Ingress controller (nginx, traefik, etc.)
- Optional: cert-manager for automatic TLS

## Quick Start

### 1. Configure Secrets

Edit `secret.yaml` and replace placeholder values:

```bash
# Generate JWT secret
JWT_SECRET=$(openssl rand -base64 32)

# Update secret.yaml with your values
sed -i "s/REPLACE_WITH_GENERATED_SECRET/$JWT_SECRET/" secret.yaml
sed -i "s/REPLACE_WITH_SECURE_PASSWORD/your-admin-password/" secret.yaml
```

**Important**: Never commit `secret.yaml` with real values to version control.

### 2. Configure Ingress

Edit `ingress.yaml`:
- Replace `cartographus.example.com` with your domain
- Update `ingressClassName` if not using nginx

### 3. Deploy

Using kubectl:
```bash
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f ingress.yaml
```

Using Kustomize:
```bash
kubectl apply -k .
```

### 4. Verify

```bash
# Check pod status
kubectl get pods -n cartographus

# Check logs
kubectl logs -n cartographus -l app.kubernetes.io/name=cartographus

# Check service
kubectl get svc -n cartographus

# Test health endpoint
kubectl port-forward -n cartographus svc/cartographus 3857:80
curl http://localhost:3857/api/health
```

## Configuration

### ConfigMap (`configmap.yaml`)

Non-sensitive configuration values:
- `HTTP_PORT`: Server port (default: 3857)
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `AUTH_MODE`: Authentication mode (jwt, basic, oidc, plex, none)
- Media server integrations (Tautulli, Plex, Jellyfin, Emby)

### Secret (`secret.yaml`)

Sensitive values:
- `JWT_SECRET`: Token signing secret (required for jwt auth)
- `ADMIN_PASSWORD`: Admin account password
- API keys for media server integrations

### Customization

Use Kustomize overlays for environment-specific configuration:

```
deploy/kubernetes/
├── base/                    # Base manifests
│   ├── kustomization.yaml
│   └── ...
├── overlays/
│   ├── development/
│   │   └── kustomization.yaml
│   ├── staging/
│   │   └── kustomization.yaml
│   └── production/
│       └── kustomization.yaml
```

Example overlay for production:
```yaml
# overlays/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base
patches:
  - patch: |
      - op: replace
        path: /spec/replicas
        value: 3
    target:
      kind: Deployment
      name: cartographus
images:
  - name: ghcr.io/tomtom215/cartographus
    newTag: v1.2.3
```

## Storage

The deployment uses a PersistentVolumeClaim for data storage:
- Database files
- Cache data
- Backup files

Adjust `storageClassName` in `deployment.yaml` for your cluster.

## TLS/HTTPS

### With cert-manager

1. Install cert-manager
2. Create ClusterIssuer for Let's Encrypt
3. Uncomment cert-manager annotations in `ingress.yaml`

### Manual TLS

Create a TLS secret:
```bash
kubectl create secret tls cartographus-tls \
  --cert=path/to/cert.pem \
  --key=path/to/key.pem \
  -n cartographus
```

## Monitoring

Cartographus exposes Prometheus metrics at `/metrics`:

```yaml
# ServiceMonitor for Prometheus Operator
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: cartographus
  namespace: cartographus
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: cartographus
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
```

## Troubleshooting

### Pod not starting

```bash
kubectl describe pod -n cartographus -l app.kubernetes.io/name=cartographus
kubectl logs -n cartographus -l app.kubernetes.io/name=cartographus --previous
```

### Health check failing

```bash
kubectl exec -n cartographus -it deploy/cartographus -- wget -qO- http://localhost:3857/api/health
```

### Database issues

```bash
# Check persistent volume
kubectl get pvc -n cartographus

# Access pod shell
kubectl exec -n cartographus -it deploy/cartographus -- sh
```

## Cleanup

```bash
kubectl delete -k .
# Or
kubectl delete namespace cartographus
```
